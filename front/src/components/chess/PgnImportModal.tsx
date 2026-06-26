import styles from "./ChessShared.module.css";

interface PgnImportModalProps {
  pgnText: string;
  onPgnTextChange: (text: string) => void;
  pgnError: string | null;
  onLoad: () => void;
  onClose: () => void;
}

export const PgnImportModal = ({
  pgnText,
  onPgnTextChange,
  pgnError,
  onLoad,
  onClose,
}: PgnImportModalProps) => (
  <div className={styles.pgnOverlay} onClick={onClose}>
    <div className={styles.pgnModal} onClick={(e) => e.stopPropagation()}>
      <div className={styles.pgnModalHeader}>
        <span className={styles.pgnModalTitle}>Importar PGN</span>
        <button className={styles.pgnCloseBtn} onClick={onClose}>
          ✕
        </button>
      </div>
      <p className={styles.pgnHint}>
        Cole a notação PGN de uma partida para visualizá-la lance a lance.
      </p>
      <textarea
        className={styles.pgnTextarea}
        value={pgnText}
        onChange={(e) => {
          onPgnTextChange(e.target.value);
        }}
        placeholder="[Event &quot;...&quot;]&#10;1. e4 e5 2. Nf3 Nc6 ..."
        rows={10}
        autoFocus
      />
      {pgnError && <div className={styles.pgnError}>{pgnError}</div>}
      <div className={styles.pgnModalActions}>
        <button className={styles.actionButton} onClick={onClose}>
          Cancelar
        </button>
        <button
          className={styles.pgnLoadButton}
          onClick={onLoad}
          disabled={!pgnText.trim()}
        >
          Carregar
        </button>
      </div>
    </div>
  </div>
);