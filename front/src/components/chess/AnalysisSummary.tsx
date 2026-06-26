import type { AiAnalysisResult } from "@/wasm/generated/wasm-contract";
import { squareName } from "@/utils/chessNotation";
import styles from "./ChessShared.module.css";

interface AnalysisSummaryProps {
  analysis: AiAnalysisResult;
  onClose?: () => void;
}

export const AnalysisSummary = ({ analysis, onClose }: AnalysisSummaryProps) => (
  <div className={styles.analysisPanel}>
    <div className={styles.analysisRow}>
      <span className={styles.analysisLabel}>Avaliação</span>
      <span className={styles.analysisValue}>
        {(analysis.score / 100).toFixed(2)}
      </span>
    </div>
    <div className={styles.analysisRow}>
      <span className={styles.analysisLabel}>Melhor lance</span>
      <span className={styles.analysisValue}>
        {squareName(analysis.from)}→{squareName(analysis.to)}
      </span>
    </div>
    <div className={styles.analysisRow}>
      <span className={styles.analysisLabel}>Profundidade</span>
      <span className={styles.analysisValue}>{analysis.depth}</span>
    </div>
    {onClose && (
      <button className={styles.closeAnalysis} onClick={onClose}>
        ✕
      </button>
    )}
  </div>
);