import type { ChessBoard as ChessBoardState, ChessPiece } from "@/types/chess";
import { getPiece } from "@/types/chess";
import styles from "./ChessBoard.module.css";

interface ChessBoardProps {
  board: ChessBoardState;
  selectedSquare?: number | null;
  validMoveSquares?: number[];
  onSquareClick?: (index: number) => Promise<void>;
  flipped?: boolean;
}

const COLOR_PREFIX = { white: "w", black: "b" } as const;
const TYPE_CODE = {
  pawn: "P",
  rook: "R",
  knight: "N",
  bishop: "B",
  queen: "Q",
  king: "K",
} as const;

const pieceUrl = (piece: ChessPiece) =>
  `/pieces/cburnett/${COLOR_PREFIX[piece.color]}${TYPE_CODE[piece.type]}.svg`;

const INDICES = Array.from({ length: 8 }, (_, i) => i);

export const ChessBoard = ({
  board,
  selectedSquare = null,
  validMoveSquares = [],
  onSquareClick,
  flipped = false,
}: ChessBoardProps) => {
  const rowIndices = flipped ? INDICES : [...INDICES].reverse();
  const colIndices = flipped ? [...INDICES].reverse() : INDICES;

  return (
    <div className={styles.board}>
      {rowIndices.map((r) =>
        colIndices.map((c) => {
          const index = r * 8 + c;
          const piece = getPiece(board, index);
          const isDark = (r + c) % 2 !== 0;
          const isSelected = selectedSquare === index;
          const isValidTarget = validMoveSquares.includes(index);

          const squareClass = [
            styles.square,
            isDark ? styles.dark : styles.light,
            isSelected ? styles.selected : "",
          ]
            .filter(Boolean)
            .join(" ");

          return (
            <div
              key={index}
              className={squareClass}
              onClick={() => onSquareClick?.(index)}
            >
              {isValidTarget &&
                (piece ? (
                  <div className={styles.captureHint} />
                ) : (
                  <div className={styles.moveHint} />
                ))}
              {piece && <Piece piece={piece} />}
            </div>
          );
        })
      )}
    </div>
  );
};

const Piece = ({ piece }: { piece: ChessPiece }) => (
  <img
    className={styles.piece}
    src={pieceUrl(piece)}
    alt={`${piece.color} ${piece.type}`}
    draggable={false}
  />
);
