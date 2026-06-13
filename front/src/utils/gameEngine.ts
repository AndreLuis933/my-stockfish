import type { Board, Color, Move, PieceType } from "@/types/game";

const DIAGONALS: [number, number][] = [
  [-1, -1],
  [-1, 1],
  [1, -1],
  [1, 1],
];

const inBounds = (r: number, c: number) => r >= 0 && r < 8 && c >= 0 && c < 8;

export const initBoard = (): Board => {
  const board: Board = Array.from({ length: 8 }, () => Array(8).fill(null));

  for (let row = 0; row < 8; row++) {
    for (let col = 0; col < 8; col++) {
      if ((row + col) % 2 !== 0) {
        if (row < 3) board[row][col] = { color: "white", type: "man" };
        if (row > 4) board[row][col] = { color: "black", type: "man" };
      }
    }
  }

  return board;
};

const buildCaptures = (
  row: number,
  col: number,
  board: Board,
  color: Color,
  type: PieceType,
  capturedSoFar: [number, number][],
): Move[] => {
  const chains: Move[] = [];

  for (const [dr, dc] of DIAGONALS) {
    if (type === "man") {
      const r1 = row + dr;
      const c1 = col + dc;
      if (!inBounds(r1, c1)) continue;

      const r2 = row + 2 * dr;
      const c2 = col + 2 * dc;
      const middle = board[r1][c1];

      if (
        inBounds(r2, c2) &&
        middle !== null &&
        middle.color !== color &&
        board[r2][c2] === null
      ) {
        const newCaptured: [number, number][] = [...capturedSoFar, [r1, c1]];
        const nextBoard = board.map((r) => [...r]);
        nextBoard[r1][c1] = null;

        const furtherCaptures = buildCaptures(
          r2,
          c2,
          nextBoard,
          color,
          type,
          newCaptured,
        );

        if (furtherCaptures.length === 0) {
          chains.push({ to: [r2, c2], captured: newCaptured });
        } else {
          chains.push(...furtherCaptures);
        }
      }
    } else {
      // Flying king: slide along the diagonal until hitting a piece
      let r = row + dr;
      let c = col + dc;

      while (inBounds(r, c)) {
        const cell = board[r][c];

        if (cell === null) {
          r += dr;
          c += dc;
          continue;
        }

        if (cell.color === color) break; // blocked by friendly piece

        // Enemy found — remove it and collect all empty landing squares beyond
        const capturedSquare: [number, number] = [r, c];
        const nextBoard = board.map((row) => [...row]);
        nextBoard[r][c] = null;

        let lr = r + dr;
        let lc = c + dc;

        while (inBounds(lr, lc) && nextBoard[lr][lc] === null) {
          const newCaptured: [number, number][] = [
            ...capturedSoFar,
            capturedSquare,
          ];
          const furtherCaptures = buildCaptures(
            lr,
            lc,
            nextBoard,
            color,
            type,
            newCaptured,
          );

          if (furtherCaptures.length === 0) {
            chains.push({ to: [lr, lc], captured: newCaptured });
          } else {
            chains.push(...furtherCaptures);
          }

          lr += dr;
          lc += dc;
        }

        break; // only one enemy per diagonal direction
      }
    }
  }

  return chains;
};

const pieceSimpleMoves = (row: number, col: number, board: Board): Move[] => {
  const piece = board[row][col];
  if (!piece) return [];

  if (piece.type === "king") {
    const moves: Move[] = [];
    for (const [dr, dc] of DIAGONALS) {
      let r = row + dr;
      let c = col + dc;
      while (inBounds(r, c) && board[r][c] === null) {
        moves.push({ to: [r, c], captured: [] });
        r += dr;
        c += dc;
      }
    }
    return moves;
  }

  const direction = piece.color === "white" ? 1 : -1;
  const moves: Move[] = [];
  for (const [dr, dc] of DIAGONALS) {
    const r1 = row + dr;
    const c1 = col + dc;
    if (inBounds(r1, c1) && dr === direction && board[r1][c1] === null) {
      moves.push({ to: [r1, c1], captured: [] });
    }
  }
  return moves;
};

export interface TurnState {
  selectable: [number, number][];
  globalMax: number;
}

export type GameResult = "white-wins" | "black-wins" | "draw" | null;

export const checkResult = (
  turnState: TurnState,
  currentTurn: Color,
  movesSinceCapture: number,
): GameResult => {
  if (turnState.selectable.length === 0) {
    return currentTurn === "white" ? "black-wins" : "white-wins";
  }
  if (movesSinceCapture >= 40) {
    return "draw";
  }
  return null;
};

export const computeTurnState = (board: Board, color: Color): TurnState => {
  let globalMax = 0;
  const captureMap: { pos: [number, number]; max: number }[] = [];

  for (let row = 0; row < 8; row++) {
    for (let col = 0; col < 8; col++) {
      const piece = board[row][col];
      if (!piece || piece.color !== color) continue;
      const chains = buildCaptures(row, col, board, piece.color, piece.type, []);
      if (chains.length === 0) continue;
      const max = Math.max(...chains.map((m) => m.captured.length));
      globalMax = Math.max(globalMax, max);
      captureMap.push({ pos: [row, col], max });
    }
  }

  if (globalMax > 0) {
    return {
      globalMax,
      selectable: captureMap.filter((e) => e.max === globalMax).map((e) => e.pos),
    };
  }

  const selectable: [number, number][] = [];
  for (let row = 0; row < 8; row++) {
    for (let col = 0; col < 8; col++) {
      const piece = board[row][col];
      if (!piece || piece.color !== color) continue;
      if (pieceSimpleMoves(row, col, board).length > 0) {
        selectable.push([row, col]);
      }
    }
  }
  return { globalMax: 0, selectable };
};

export const validMoves = (row: number, col: number, board: Board, globalMax: number): Move[] => {
  const piece = board[row][col];
  if (!piece) return [];

  const captures = buildCaptures(row, col, board, piece.color, piece.type, []);
  const bestCaptures = captures.filter((m) => m.captured.length === globalMax);

  return bestCaptures.length > 0 ? bestCaptures : pieceSimpleMoves(row, col, board);
};

export const applyMove = (
  from: [number, number],
  move: Move,
  board: Board,
): Board => {
  const next = board.map((row) => [...row]);
  const [fromR, fromC] = from;
  const [toR, toC] = move.to;

  next[toR][toC] = next[fromR][fromC];
  next[fromR][fromC] = null;

  for (const [capR, capC] of move.captured) {
    next[capR][capC] = null;
  }

  // Promote man to king on reaching the opposite back rank
  const piece = next[toR][toC];
  if (piece?.type === "man") {
    const backRank = piece.color === "white" ? 7 : 0;
    if (toR === backRank) {
      next[toR][toC] = { ...piece, type: "king" };
    }
  }

  return next;
};
