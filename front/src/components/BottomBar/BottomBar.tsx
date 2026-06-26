import styles from "./BottomBar.module.css";

interface BottomBarProps {
  fen: string;
  pgn: string;
  onJumpToStart: () => void;
  onJumpToEnd: () => void;
  onStepBack: () => void;
  onStepForward: () => void;
  canStepBack: boolean;
  canStepForward: boolean;
}

export const BottomBar = ({
  fen,
  pgn,
  onJumpToStart,
  onJumpToEnd,
  onStepBack,
  onStepForward,
  canStepBack,
  canStepForward,
}: BottomBarProps) => {
  return (
    <div className={styles.bar}>
      <div className={styles.fieldGroup}>
        <span className={styles.fieldLabel}>FEN</span>
        <div className={styles.fieldRow}>
          <input
            className={styles.fieldInput}
            value={fen}
            readOnly
            placeholder="FEN da posição atual"
          />
          <div className={styles.controls}>
            <button
              className={styles.controlBtn}
              onClick={onJumpToStart}
              disabled={!canStepBack}
              title="Início"
            >
              |◀◀
            </button>
            <button
              className={styles.controlBtn}
              onClick={onStepBack}
              disabled={!canStepBack}
              title="Anterior"
            >
              ◀
            </button>
            <button
              className={styles.controlBtn}
              onClick={onStepForward}
              disabled={!canStepForward}
              title="Próximo"
            >
              ▶
            </button>
            <button
              className={styles.controlBtn}
              onClick={onJumpToEnd}
              disabled={!canStepForward}
              title="Fim"
            >
              ▶|
            </button>
            <button className={`${styles.controlBtn} ${styles.menuBtn}`} title="Menu">
              ☰
            </button>
          </div>
        </div>
      </div>

      <div className={styles.fieldGroup}>
        <span className={styles.fieldLabel}>PGN</span>
        <textarea
          className={`${styles.fieldInput} ${styles.fieldInputTextarea}`}
          value={pgn}
          readOnly
          placeholder="PGN da partida"
          rows={3}
        />
      </div>
    </div>
  );
};