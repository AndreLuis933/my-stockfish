import type { Board, Color, Move } from "@/types/game";
import { applyMove, checkResult, computeTurnState, validMoves } from "@/utils/gameEngine";

export interface AIMove {
  from: [number, number];
  move: Move;
}

export interface SearchResult {
  move: AIMove;
  depth: number;
  score: number;
  nodes: number;
  timeMs: number;
}

interface SearchCtx {
  startTime: number;
  timeLimitMs: number;
  nodes: number;
  aborted: boolean;
}

const PIECE_VALUE = { man: 100, king: 300 };
const WIN = 100_000;

const flip = (color: Color): Color => (color === "white" ? "black" : "white");

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
      score += piece.color === color ? base + advance : -(base + advance);
    }
  }
  return score;
};

// Captures-first, then try the best move from the previous IDDFS iteration first
const orderMoves = (moves: AIMove[], previousBest: AIMove | null = null): AIMove[] => {
  const sorted = [...moves].sort((a, b) => b.move.captured.length - a.move.captured.length);
  if (!previousBest) return sorted;
  const idx = sorted.findIndex(
    (m) =>
      m.from[0] === previousBest.from[0] &&
      m.from[1] === previousBest.from[1] &&
      m.move.to[0] === previousBest.move.to[0] &&
      m.move.to[1] === previousBest.move.to[1],
  );
  if (idx > 0) sorted.unshift(...sorted.splice(idx, 1));
  return sorted;
};

const minimax = (
  board: Board,
  aiColor: Color,
  currentColor: Color,
  depth: number,
  alpha: number,
  beta: number,
  movesSinceCapture: number,
  ctx: SearchCtx,
): number => {
  ctx.nodes++;
  // Check the clock every 2048 nodes to avoid paying for Date.now() on every call
  if ((ctx.nodes & 2047) === 0 && performance.now() - ctx.startTime >= ctx.timeLimitMs) {
    ctx.aborted = true;
  }
  if (ctx.aborted) return 0;

  const turnState = computeTurnState(board, currentColor);
  const result = checkResult(turnState, currentColor, movesSinceCapture);
  if (result !== null) {
    if (result === "draw") return 0;
    return result === `${aiColor}-wins` ? WIN : -WIN;
  }
  if (depth === 0) return evaluateBoard(board, aiColor);

  const moves = orderMoves(getAllMoves(board, currentColor));
  const isMaximizing = currentColor === aiColor;
  let best = isMaximizing ? -Infinity : Infinity;

  for (const { from, move } of moves) {
    if (ctx.aborted) break;
    const nextBoard = applyMove(from, move, board);
    const movedPiece = board[from[0]][from[1]];
    const resets = move.captured.length > 0 || movedPiece?.type === "man";
    const score = minimax(
      nextBoard,
      aiColor,
      flip(currentColor),
      depth - 1,
      alpha,
      beta,
      resets ? 0 : movesSinceCapture + 1,
      ctx,
    );

    if (isMaximizing) {
      if (score > best) best = score;
      if (best > alpha) alpha = best;
    } else {
      if (score < best) best = score;
      if (best < beta) beta = best;
    }
    if (beta <= alpha) break;
  }

  return best;
};

// Searches a single depth. Returns null if the search was aborted mid-way (partial = unreliable).
const searchAtDepth = (
  board: Board,
  color: Color,
  movesSinceCapture: number,
  depth: number,
  ctx: SearchCtx,
  previousBest: AIMove | null,
): { move: AIMove; score: number } | null => {
  const moves = orderMoves(getAllMoves(board, color), previousBest);
  if (moves.length === 0) return null;

  let bestScore = -Infinity;
  let bestMove = moves[0];

  for (const candidate of moves) {
    if (ctx.aborted) return null;
    const nextBoard = applyMove(candidate.from, candidate.move, board);
    const movedPiece = board[candidate.from[0]][candidate.from[1]];
    const resets = candidate.move.captured.length > 0 || movedPiece?.type === "man";
    const score = minimax(
      nextBoard,
      color,
      flip(color),
      depth - 1,
      -Infinity,
      Infinity,
      resets ? 0 : movesSinceCapture + 1,
      ctx,
    );

    if (!ctx.aborted && score > bestScore) {
      bestScore = score;
      bestMove = candidate;
    }
  }

  return ctx.aborted ? null : { move: bestMove, score: bestScore };
};

// Fixed-depth search — baseline for benchmarking a specific depth.
export const pickBestMove = (
  board: Board,
  color: Color,
  movesSinceCapture: number,
  depth = 8,
): AIMove | null => {
  const ctx: SearchCtx = { startTime: 0, timeLimitMs: Infinity, nodes: 0, aborted: false };
  return searchAtDepth(board, color, movesSinceCapture, depth, ctx, null)?.move ?? null;
};

// Iterative deepening — deepens until the time budget runs out, then returns the best
// fully-completed depth. Partial results from an aborted depth are discarded.
export const pickBestMoveWithTime = (
  board: Board,
  color: Color,
  movesSinceCapture: number,
  timeLimitMs = 200,
): SearchResult | null => {
  const moves = getAllMoves(board, color);
  if (moves.length === 0) return null;

  const ctx: SearchCtx = {
    startTime: performance.now(),
    timeLimitMs,
    nodes: 0,
    aborted: false,
  };

  let best: SearchResult = {
    move: moves[0],
    depth: 0,
    score: evaluateBoard(board, color),
    nodes: 0,
    timeMs: 0,
  };

  let previousBest: AIMove | null = null;

  for (let depth = 1; depth <= 32; depth++) {
    const result = searchAtDepth(board, color, movesSinceCapture, depth, ctx, previousBest);
    if (ctx.aborted || result === null) break;

    previousBest = result.move;
    best = {
      move: result.move,
      depth,
      score: result.score,
      nodes: ctx.nodes,
      timeMs: performance.now() - ctx.startTime,
    };

    if (Math.abs(result.score) >= WIN) break; // forced win/loss — no need to go deeper
  }

  return best;
};
