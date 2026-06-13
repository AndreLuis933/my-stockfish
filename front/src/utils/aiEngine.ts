import type { Board, Color, Move } from "@/types/game";
import { applyMove, checkResult, computeTurnState, validMoves } from "@/utils/gameEngine";

export interface AIMove {
  from: [number, number];
  move: Move;
}

const PIECE_VALUE = { man: 100, king: 300 };

const getAllMoves = (board: Board, color: Color): AIMove[] => {
  const turnState = computeTurnState(board, color);
  const moves: AIMove[] = [];
  for (const [r, c] of turnState.selectable) {
    for (const move of validMoves(r, c, board, turnState.globalMax)) {
      moves.push({ from: [r, c], move });
    }
  }
  return moves;
};

const evaluateBoard = (board: Board, color: Color): number => {
  let score = 0;
  for (let r = 0; r < 8; r++) {
    for (let c = 0; c < 8; c++) {
      const piece = board[r][c];
      if (!piece) continue;
      const base = PIECE_VALUE[piece.type];
      const advance = piece.type === "man" ? (piece.color === "white" ? r * 4 : (7 - r) * 4) : 0;
      const value = base + advance;
      score += piece.color === color ? value : -value;
    }
  }
  return score;
};

const nextColor = (color: Color): Color => (color === "white" ? "black" : "white");

const minimax = (
  board: Board,
  aiColor: Color,
  currentColor: Color,
  depth: number,
  alpha: number,
  beta: number,
  movesSinceCapture: number,
): number => {
  const turnState = computeTurnState(board, currentColor);
  const result = checkResult(turnState, currentColor, movesSinceCapture);

  if (result !== null) {
    if (result === "draw") return 0;
    return result === `${aiColor}-wins` ? 100_000 : -100_000;
  }

  if (depth === 0) return evaluateBoard(board, aiColor);

  const moves = getAllMoves(board, currentColor);
  const isMaximizing = currentColor === aiColor;
  let best = isMaximizing ? -Infinity : Infinity;

  for (const { from, move } of moves) {
    const nextBoard = applyMove(from, move, board);
    const movedPiece = board[from[0]][from[1]];
    const resets = move.captured.length > 0 || movedPiece?.type === "man";
    const nextMSC = resets ? 0 : movesSinceCapture + 1;
    const score = minimax(nextBoard, aiColor, nextColor(currentColor), depth - 1, alpha, beta, nextMSC);

    if (isMaximizing) {
      best = Math.max(best, score);
      alpha = Math.max(alpha, best);
    } else {
      best = Math.min(best, score);
      beta = Math.min(beta, best);
    }

    if (beta <= alpha) break;
  }

  return best;
};

export const pickBestMove = (
  board: Board,
  color: Color,
  movesSinceCapture: number,
  depth = 6,
): AIMove | null => {
  const moves = getAllMoves(board, color);
  if (moves.length === 0) return null;

  let best = -Infinity;
  let bestMove = moves[0];

  for (const candidate of moves) {
    const nextBoard = applyMove(candidate.from, candidate.move, board);
    const movedPiece = board[candidate.from[0]][candidate.from[1]];
    const resets = candidate.move.captured.length > 0 || movedPiece?.type === "man";
    const nextMSC = resets ? 0 : movesSinceCapture + 1;
    const score = minimax(nextBoard, color, nextColor(color), depth - 1, -Infinity, Infinity, nextMSC);
    if (score > best) {
      best = score;
      bestMove = candidate;
    }
  }

  return bestMove;
};
