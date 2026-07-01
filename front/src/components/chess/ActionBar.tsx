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
  showBookMoves: boolean;
  onToggleBookMoves: () => void;
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
  showBookMoves,
  onToggleBookMoves,
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
    <button
      className={`${styles.actionButton} ${showBookMoves ? styles.actionButtonActive : ""}`}
      onClick={onToggleBookMoves}
      title="Mostra os lances do livro de aberturas para a posição atual"
    >
      {showBookMoves ? "Livro ✓" : "Livro"}
    </button>
  </div>
);