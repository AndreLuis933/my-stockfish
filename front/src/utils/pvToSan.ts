import type { ChessBoard, ChessColor } from "@/types/chess";
import { toSan, type MoveData } from "@/utils/chessNotation";

export interface PvSanEntry {
  san: string;
  color: ChessColor;
}

export const pvToSan = (
  boardBefore: ChessBoard,
  pv: { from: number; to: number; promotion?: number }[],
  startColor: ChessColor,
): PvSanEntry[] => {
  const entries: PvSanEntry[] = [];
  const board = boardBefore.slice();
  let color = startColor;

  for (const move of pv) {
    if (!move || move.from == null || move.to == null) break;
    const moveData: MoveData = {
      from: move.from,
      to: move.to,
      promotion: move.promotion,
    };
    const san = toSan(board, moveData, color, null, false);
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