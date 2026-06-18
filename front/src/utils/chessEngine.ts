import type {
  ChessBoard,
  ChessColor,
  ChessPieceType,
} from "@/types/chess";

const WHITE_PAWN = 0b01000001;
const WHITE_ROOK = 0b01001000;
const WHITE_KNIGHT = 0b01000010;
const WHITE_BISHOP = 0b01000100;
const WHITE_QUEEN = 0b00010000 | 0b01000000;
const WHITE_KING = 0b00100000 | 0b01000000;

const BLACK_PAWN = 0b10000001;
const BLACK_ROOK = 0b10001000;
const BLACK_KNIGHT = 0b10000010;
const BLACK_BISHOP = 0b10000100;
const BLACK_QUEEN = 0b00010000 | 0b10000000;
const BLACK_KING = 0b00100000 | 0b10000000;

const EMPTY = 0;

export const initChessBoard = (): ChessBoard => {
  const board: ChessBoard = Array(64).fill(EMPTY);

  for (let col = 0; col < 8; col++) {
    board[col] = BLACK_BACK_RANK[col];
    board[8 + col] = BLACK_PAWN;
    board[48 + col] = WHITE_PAWN;
    board[56 + col] = WHITE_BACK_RANK[col];
  }

  return board;
};

const BLACK_BACK_RANK: number[] = [
  BLACK_ROOK,
  BLACK_KNIGHT,
  BLACK_BISHOP,
  BLACK_QUEEN,
  BLACK_KING,
  BLACK_BISHOP,
  BLACK_KNIGHT,
  BLACK_ROOK,
];

const WHITE_BACK_RANK: number[] = [
  WHITE_ROOK,
  WHITE_KNIGHT,
  WHITE_BISHOP,
  WHITE_QUEEN,
  WHITE_KING,
  WHITE_BISHOP,
  WHITE_KNIGHT,
  WHITE_ROOK,
];

export const bytesToChessBoard = (bytes: number[]): ChessBoard => bytes.slice();

export const emptyBoard = (): ChessBoard => Array(64).fill(EMPTY);

export const pieceByte = (
  color: ChessColor,
  type: ChessPieceType,
): number => {
  const colorBits = color === "white" ? 0b01000000 : 0b10000000;
  const typeMap: Record<ChessPieceType, number> = {
    pawn: 0b00000001,
    knight: 0b00000010,
    bishop: 0b00000100,
    rook: 0b00001000,
    queen: 0b00010000,
    king: 0b00100000,
  };
  return colorBits | typeMap[type];
};

export const squareIndex = (row: number, col: number): number => row * 8 + col;

export const squareRowCol = (index: number): [number, number] => [
  Math.floor(index / 8),
  index % 8,
];
