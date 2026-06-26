import type { ChessPieceType } from "@/types/chess";

const PIECE_LETTER_TO_TYPE: Record<string, ChessPieceType> = {
  K: "king",
  Q: "queen",
  R: "rook",
  B: "bishop",
  N: "knight",
};

export interface ParsedSan {
  pieceType: ChessPieceType | null;
  rest: string;
}

export const parseSanFigurine = (san: string): ParsedSan => {
  const clean = san.replace(/[+#]+$/, "").replace(/[!?]+$/, "");
  const first = clean[0];
  if (first && PIECE_LETTER_TO_TYPE[first]) {
    return { pieceType: PIECE_LETTER_TO_TYPE[first], rest: clean.slice(1) };
  }
  return { pieceType: null, rest: clean };
};