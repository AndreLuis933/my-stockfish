import type { AiAnalysisResult } from "@/wasm/generated/wasm-contract";

export type ChessColor = "white" | "black";
export type ChessPieceType = "pawn" | "rook" | "knight" | "bishop" | "queen" | "king";

export interface ChessPiece {
  color: ChessColor;
  type: ChessPieceType;
}

export type ChessBoard = number[];

export interface ChessMove {
  from: number;
  to: number;
  promotion?: number;
}

export interface HistoryEntry {
  san: string;
  color: ChessColor;
  from: number;
  to: number;
  promotion?: number;
  boardBefore: ChessBoard;
  boardAfter: ChessBoard;
  checkSquareAfter: number | null;
  isCheckmate: boolean;
  analysis?: AiAnalysisResult | null;
}

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

const PIECE_TYPE_BITS: Record<ChessPieceType, number> = {
  pawn: 0b00000001,
  knight: 0b00000010,
  bishop: 0b00000100,
  rook: 0b00001000,
  queen: 0b00010000,
  king: 0b00100000,
};

export const colorBits = (color: ChessColor): number =>
  color === "white" ? WHITE_BITS : 0b10000000;

export const pieceByte = (color: ChessColor, type: ChessPieceType): number =>
  colorBits(color) | PIECE_TYPE_BITS[type];

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

export const decodePieceByte = (byte: number): ChessPiece => {
  const typeBits = byte & 0b00111111;
  const type = PIECE_TYPES[typeBits];
  const color: ChessColor =
    (byte & COLOR_BITS) === WHITE_BITS ? "white" : "black";
  return { color, type };
};

export const emptyBoard = (): ChessBoard => Array(64).fill(0);