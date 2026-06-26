import styles from "./ChessShared.module.css";

interface ActionBarProps {
  flipped: boolean;
  onFlip: () => void;
  onRestart: () => void;
  copyStatus: "idle" | "copied";
  onCopyPgn: () => void;
  onPastePgn: () => void;
  canAnalyze: boolean;
  analyzing: boolean;
  onAnalyze: () => void;
  autoAnalyze: boolean;
  onToggleAutoAnalyze: () => void;
  showAnalysisToggle: boolean;
  analysisEnabled: boolean;
  onToggleAnalysis: () => void;
  hasHistory: boolean;
}

export const ActionBar = ({
  flipped,
  onFlip,
  onRestart,
  copyStatus,
  onCopyPgn,
  onPastePgn,
  canAnalyze,
  analyzing,
  onAnalyze,
  autoAnalyze,
  onToggleAutoAnalyze,
  showAnalysisToggle,
  analysisEnabled,
  onToggleAnalysis,
  hasHistory,
}: ActionBarProps) => (
  <div className={styles.actions}>
    <button className={styles.actionButton} onClick={onRestart}>
      Reiniciar
    </button>
    <button
      className={`${styles.actionButton} ${flipped ? styles.actionButtonActive : ""}`}
      onClick={onFlip}
    >
      Girar ↺
    </button>
    {hasHistory && (
      <button
        className={`${styles.actionButton} ${copyStatus === "copied" ? styles.actionButtonActive : ""}`}
        onClick={onCopyPgn}
        title="Copia o PGN da partida para a área de transferência"
      >
        {copyStatus === "copied" ? "Copiado ✓" : "Copiar PGN"}
      </button>
    )}
    <button
      className={styles.actionButton}
      onClick={onPastePgn}
      title="Cola um PGN para visualizar a partida"
    >
      Colar PGN
    </button>
    {canAnalyze && (
      <button
        className={styles.actionButton}
        onClick={onAnalyze}
        disabled={analyzing}
      >
        {analyzing ? "Analisando..." : "Analisar"}
      </button>
    )}
    {showAnalysisToggle && (
      <button
        className={`${styles.actionButton} ${autoAnalyze ? styles.actionButtonActive : ""}`}
        onClick={onToggleAutoAnalyze}
        title="Analisa automaticamente cada posição após cada lance"
      >
        {autoAnalyze ? "Auto ✓" : "Analisar auto"}
      </button>
    )}
    {showAnalysisToggle && (
      <button
        className={`${styles.actionButton} ${analysisEnabled ? styles.actionButtonActive : ""}`}
        onClick={onToggleAnalysis}
        title={analysisEnabled ? "Parar análise contínua" : "Iniciar análise contínua"}
      >
        {analysisEnabled ? "Análise ⏸" : "Análise ▶"}
      </button>
    )}
  </div>
);