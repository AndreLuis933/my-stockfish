import type { Board as BoardState, Cell } from "@/types/game";
import styles from "./Board.module.css";

interface BoardProps {
  board: BoardState;
  selectedSquare?: [number, number] | null;
  validMoveSquares?: [number, number][];
  mustMoveSquares?: [number, number][];
  onSquareClick?: (row: number, col: number) => void;
  flipped?: boolean;
}

const INDICES = Array.from({ length: 8 }, (_, i) => i);

export const Board = ({
  board,
  selectedSquare = null,
  validMoveSquares = [],
  mustMoveSquares = [],
  onSquareClick,
  flipped = false,
}: BoardProps) => {
  const rowIndices = flipped ? [...INDICES].reverse() : INDICES;
  const colIndices = flipped ? [...INDICES].reverse() : INDICES;

  return (
    <div className={styles.board}>
      {rowIndices.map((r) =>
        colIndices.map((c) => {
          const cell = board[r][c];
          const isDark = (r + c) % 2 !== 0;
          const isSelected = selectedSquare?.[0] === r && selectedSquare?.[1] === c;
          const isValidTarget = validMoveSquares.some(([vr, vc]) => vr === r && vc === c);
          const isMustMove = mustMoveSquares.some(([mr, mc]) => mr === r && mc === c);

          const squareClass = [
            styles.square,
            isDark ? styles.dark : styles.light,
            isSelected ? styles.selected : "",
          ]
            .filter(Boolean)
            .join(" ");

          return (
            <div
              key={`${r}-${c}`}
              className={squareClass}
              onClick={() => onSquareClick?.(r, c)}
            >
              {isValidTarget && (cell ? (
                <div className={styles.captureHint} />
              ) : (
                <div className={styles.moveHint} />
              ))}

              {isMustMove && <div className={styles.mustMoveHint} />}

              {cell && <Piece cell={cell} />}
            </div>
          );
        })
      )}
    </div>
  );
};

const Piece = ({ cell }: { cell: NonNullable<Cell> }) => {
  const pieceClass = [
    styles.piece,
    cell.color === "white" ? styles.white : styles.black,
  ].join(" ");

  return (
    <div className={pieceClass}>
      {cell.type === "king" && <div className={styles.kingMark} />}
    </div>
  );
};
