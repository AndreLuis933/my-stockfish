import type { Board, Move } from "@/types/game";

const DIAGONALS: [number, number][] = [
  [-1, -1], [-1, 1],
  [1, -1],  [1, 1],
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

export const validMoves = (row: number, col: number, board: Board): Move[] => {
  const piece = board[row][col];
  if (!piece) return [];

  const direction = piece.color === "white" ? 1 : -1;
  const simpleMoves: Move[] = [];
  const captures: Move[] = [];

  for (const [dr, dc] of DIAGONALS) {
    const r1 = row + dr;
    const c1 = col + dc;
    if (!inBounds(r1, c1)) continue;

    // Simple move: only in the forward direction, destination must be empty
    if (dr === direction && board[r1][c1] === null) {
      simpleMoves.push({ to: [r1, c1], captured: null });
    }

    // Capture: middle square has an enemy, landing square is empty
    const r2 = row + 2 * dr;
    const c2 = col + 2 * dc;
    const middle = board[r1][c1];

    if (
      inBounds(r2, c2) &&
      middle !== null &&
      middle.color !== piece.color &&
      board[r2][c2] === null
    ) {
      captures.push({ to: [r2, c2], captured: [r1, c1] });
    }
  }

  // Captures are mandatory — return them exclusively when available
  return captures.length > 0 ? captures : simpleMoves;
};

export const applyMove = (from: [number, number], move: Move, board: Board): Board => {
  const next = board.map((row) => [...row]);
  const [fromR, fromC] = from;
  const [toR, toC] = move.to;

  next[toR][toC] = next[fromR][fromC];
  next[fromR][fromC] = null;

  if (move.captured) {
    const [capR, capC] = move.captured;
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
