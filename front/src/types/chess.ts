export type ChessColor = "white" | "black";
export type ChessPieceType = "pawn" | "rook" | "knight" | "bishop" | "queen" | "king";

export interface ChessPiece {
  color: ChessColor;
  type: ChessPieceType;
}

export type ChessBoard = number[];

const COLOR_BITS = 0b11000000;
const WHITE_BITS = 0b01000000;

const PIECE_TYPES: Record<number, ChessPieceType> = {
  0b00000001: "pawn",
  0b00000010: "knight",
  0b00000100: "bishop",
  0b00001000: "rook",
  0b00010000: "queen",
  0b00100000: "king",
};

export const getPiece = (board: ChessBoard, index: number): ChessPiece | null => {
  const byte = board[index];
  if (!byte) return null;

  const typeBits = byte & 0b00111111;
  const type = PIECE_TYPES[typeBits];
  if (!type) return null;

  const color: ChessColor =
    (byte & COLOR_BITS) === WHITE_BITS ? "white" : "black";

  return { color, type };
};

export const getPieceAt = (
  board: ChessBoard,
  row: number,
  col: number,
): ChessPiece | null => getPiece(board, row * 8 + col);

export const squareIndex = (row: number, col: number): number => row * 8 + col;
export const squareRowCol = (index: number): [number, number] => [
  Math.floor(index / 8),
  index % 8,
];
