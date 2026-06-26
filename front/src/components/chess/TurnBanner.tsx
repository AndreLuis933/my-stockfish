import type { ChessColor } from "@/types/chess";
import type { ChessResult } from "@/pages/chess/Chess.types";
import styles from "./ChessShared.module.css";

interface TurnBannerProps {
  currentPlayer: ChessColor;
  result: ChessResult;
  resultText: string | null;
  aiThinking: boolean;
  checkSquare: number | null;
  isAtLatest: boolean;
}

export const TurnBanner = ({
  currentPlayer,
  result,
  resultText,
  aiThinking,
  checkSquare,
  isAtLatest,
}: TurnBannerProps) => (
  <div className={styles.turnBanner}>
    <div
      className={`${styles.dot} ${styles.dotWhite} ${currentPlayer === "white" ? styles.active : ""}`}
    />
    <span className={styles.turnText}>
      {result !== null && resultText ? (
        resultText
      ) : (
        <>
          {currentPlayer === "white" ? "Vez das Brancas" : "Vez das Pretas"}
          {aiThinking && <span className={styles.thinkingBadge}>IA pensando...</span>}
          {checkSquare !== null && <span className={styles.checkBadge}>Xeque!</span>}
          {!isAtLatest && <span className={styles.historyBadge}> revisitando</span>}
        </>
      )}
    </span>
    <div
      className={`${styles.dot} ${styles.dotBlack} ${currentPlayer === "black" ? styles.active : ""}`}
    />
  </div>
);