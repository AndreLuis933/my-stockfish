import { Figurine } from "@/components/Figurine/Figurine";
import type { ChessColor, ChessPieceType } from "@/types/chess";
import { parseSanFigurine } from "@/utils/chessFigurine";

interface SanTextProps {
  san: string;
  color: ChessColor;
  figurineClassName?: string;
  textClassName?: string;
  className?: string;
}

export const SanText = ({
  san,
  color,
  figurineClassName,
  textClassName,
  className,
}: SanTextProps) => {
  const { pieceType, rest }: { pieceType: ChessPieceType | null; rest: string } =
    parseSanFigurine(san);
  return (
    <span className={className}>
      {pieceType && (
        <Figurine type={pieceType} color={color} className={figurineClassName} />
      )}
      <span className={textClassName}>{rest}</span>
    </span>
  );
};