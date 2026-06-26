import type { ChessColor, ChessPieceType } from "@/types/chess";
import { pieceImageUrl } from "@/utils/chessAssets";

interface FigurineProps {
  type: ChessPieceType;
  color: ChessColor;
  className?: string;
}

export const Figurine = ({ type, color, className }: FigurineProps) => {
  if (type === "pawn") return null;
  return (
    <img
      src={pieceImageUrl({ color, type })}
      alt={type}
      className={className}
      draggable={false}
    />
  );
};