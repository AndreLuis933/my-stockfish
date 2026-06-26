import type { ChessBoard, ChessColor, ChessMove } from "@/types/chess";
import { toSan } from "@/utils/chessNotation";

export interface PvSanEntry {
  san: string;
  color: ChessColor;
}

export const pvToSan = (
  boardBefore: ChessBoard,
  pv: ChessMove[],
  startColor: ChessColor,
): PvSanEntry[] => {
  const entries: PvSanEntry[] = [];
  const board = boardBefore.slice();
  let color = startColor;

  for (const move of pv) {
    if (!move || move.from == null || move.to == null) break;
    const san = toSan(board, move, color, null, false);
    entries.push({ san, color });

    board[move.to] = board[move.from] ?? 0;
    board[move.from] = 0;
    if (move.promotion) {
      board[move.to] = move.promotion;
    }
    color = color === "white" ? "black" : "white";
  }

  return entries;
};