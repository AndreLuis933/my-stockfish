import type { ChessPiece } from "@/types/chess";

const COLOR_PREFIX = { white: "w", black: "b" } as const;
const TYPE_CODE = {
  pawn: "P",
  rook: "R",
  knight: "N",
  bishop: "B",
  queen: "Q",
  king: "K",
} as const;

export const pieceImageUrl = (piece: ChessPiece): string =>
  `/pieces/cburnett/${COLOR_PREFIX[piece.color]}${TYPE_CODE[piece.type]}.svg`;