import type { Board as BoardState, Cell } from "@/types/game";
import styles from "./Board.module.css";

interface BoardProps {
  board: BoardState;
  selectedSquare?: [number, number] | null;
  validMoveSquares?: [number, number][];
  onSquareClick?: (row: number, col: number) => void;
}

export const Board = ({
  board,
  selectedSquare = null,
  validMoveSquares = [],
  onSquareClick,
}: BoardProps) => {
  return (
    <div className={styles.board}>
      {board.map((row, r) =>
        row.map((cell, c) => {
          const isDark = (r + c) % 2 !== 0;
          const isSelected = selectedSquare?.[0] === r && selectedSquare?.[1] === c;
          const isValidTarget = validMoveSquares.some(([vr, vc]) => vr === r && vc === c);

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
