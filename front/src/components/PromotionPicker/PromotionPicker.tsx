import type { ChessPiece } from "@/types/chess";
import { decodePieceByte } from "@/types/chess";
import { pieceImageUrl } from "@/utils/chessAssets";
import styles from "./PromotionPicker.module.css";

interface PromotionPickerProps {
  options: number[];
  onSelect: (pieceByte: number) => void;
  onCancel: () => void;
}

const ORDER: ChessPiece["type"][] = ["queen", "knight", "rook", "bishop"];

export const PromotionPicker = ({
  options,
  onSelect,
  onCancel,
}: PromotionPickerProps) => {
  const pieces = options
    .map((byte) => ({ byte, piece: decodePieceByte(byte) }))
    .sort(
      (a, b) =>
        ORDER.indexOf(a.piece.type) - ORDER.indexOf(b.piece.type),
    );

  return (
    <div className={styles.overlay} onClick={onCancel}>
      <div className={styles.card} onClick={(e) => e.stopPropagation()}>
        <span className={styles.title}>Escolha a promoção</span>
        <div className={styles.options}>
          {pieces.map(({ byte, piece }) => (
            <button
              key={byte}
              className={styles.option}
              onClick={() => onSelect(byte)}
            >
              <img
                className={styles.piece}
                src={pieceImageUrl(piece)}
                alt={`${piece.color} ${piece.type}`}
                draggable={false}
              />
            </button>
          ))}
        </div>
      </div>
    </div>
  );
};