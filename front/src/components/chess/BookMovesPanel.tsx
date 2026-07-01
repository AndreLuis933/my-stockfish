import { SanText } from "@/components/chess/SanText";
import type { BookMoveEntry } from "@/wasm/generated/wasm-contract";
import type { ChessColor } from "@/types/chess";
import styles from "./ChessShared.module.css";

interface BookMovesPanelProps {
  moves: BookMoveEntry[];
  loading: boolean;
  currentPlayer: ChessColor;
  onPlayMove: (from: number, to: number, promotion?: number) => void;
  onClose: () => void;
}

export const BookMovesPanel = ({
  moves,
  loading,
  currentPlayer,
  onPlayMove,
  onClose,
}: BookMovesPanelProps) => {
  if (loading) {
    return (
      <div className={styles.bookMovesPanel}>
        <div className={styles.bookMovesHeader}>
          <span className={styles.bookMovesTitle}>Livro de Aberturas</span>
          <button className={styles.closeAnalysis} onClick={onClose}>
            ✕
          </button>
        </div>
        <div className={styles.bookMovesEmpty}>Carregando...</div>
      </div>
    );
  }

  if (moves.length === 0) {
    return (
      <div className={styles.bookMovesPanel}>
        <div className={styles.bookMovesHeader}>
          <span className={styles.bookMovesTitle}>Livro de Aberturas</span>
          <button className={styles.closeAnalysis} onClick={onClose}>
            ✕
          </button>
        </div>
        <div className={styles.bookMovesEmpty}>Sem lances no livro para esta posição</div>
      </div>
    );
  }

  const totalWeight = moves.reduce((sum, m) => sum + m.weight, 0);

  return (
    <div className={styles.bookMovesPanel}>
      <div className={styles.bookMovesHeader}>
        <span className={styles.bookMovesTitle}>
          Livro de Aberturas
        </span>
        <span className={styles.bookMovesCount}>{moves.length} lances</span>
        <button className={styles.closeAnalysis} onClick={onClose}>
          ✕
        </button>
      </div>
      <div className={styles.bookMovesList}>
        {moves.map((move, i) => {
          const pct = totalWeight > 0 ? (move.weight / totalWeight) * 100 : 0;
          return (
            <button
              key={`${move.from}-${move.to}-${move.promotion ?? 0}`}
              className={styles.bookMoveRow}
              onClick={() => onPlayMove(move.from, move.to, move.promotion)}
              title={`Jogar ${move.san}`}
            >
              <span className={styles.bookMoveIndex}>{i + 1}.</span>
              <SanText
                san={move.san}
                color={currentPlayer}
                figurineClassName={styles.bookMoveFigurine}
                textClassName={styles.bookMoveText}
              />
              <div className={styles.bookMoveWeightBar}>
                <div
                  className={styles.bookMoveWeightFill}
                  style={{ width: `${pct}%` }}
                />
              </div>
              <span className={styles.bookMovePct}>{pct.toFixed(0)}%</span>
            </button>
          );
        })}
      </div>
    </div>
  );
};