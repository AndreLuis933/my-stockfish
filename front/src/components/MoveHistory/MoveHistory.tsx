import { useEffect, useRef } from "react";
import { formatClock } from "@/hooks/useChessClock";
import { Figurine } from "@/components/Figurine/Figurine";
import type { AiAnalysisResult } from "@/wasm/generated/wasm-contract";
import type { ChessColor } from "@/types/chess";
import { formatEval } from "@/utils/chessAnalysis";
import { parseSanFigurine } from "@/utils/chessFigurine";
import styles from "./MoveHistory.module.css";

export interface HistoryEntry {
  san: string;
  color: ChessColor;
  analysis?: AiAnalysisResult | null;
}

interface MoveHistoryProps {
  history: HistoryEntry[];
  currentPly: number;
  onJump: (ply: number) => void;
  clocks: { white: number; black: number };
  activeColor: ChessColor;
  clockEnabled: boolean;
  flagFallen: ChessColor | null;
  result: string | null;
  resultText: string | null;
  onRestart: () => void;
}

const NAV_BUTTONS = [
  { delta: -Infinity, label: "|◀", title: "Início" },
  { delta: -1, label: "◀", title: "Anterior" },
  { delta: 1, label: "▶", title: "Próximo" },
  { delta: Infinity, label: "▶|", title: "Fim" },
] as const;

const renderMoveText = (san: string, color: ChessColor): React.ReactNode => {
  const { pieceType, rest } = parseSanFigurine(san);
  return (
    <>
      {pieceType && (
        <Figurine
          type={pieceType}
          color={color}
          className={styles.moveFigurine}
        />
      )}
      <span className={styles.moveText}>{rest}</span>
    </>
  );
};

export const MoveHistory = ({
  history,
  currentPly,
  onJump,
  clocks,
  activeColor,
  clockEnabled,
  flagFallen,
  result,
  resultText,
  onRestart,
}: MoveHistoryProps) => {
  const listRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const el = listRef.current?.querySelector(
      `[data-ply="${currentPly}"]`,
    ) as HTMLElement | null;
    el?.scrollIntoView({ block: "nearest", behavior: "smooth" });
  }, [currentPly]);

  const rows: { num: number; white?: HistoryEntry; black?: HistoryEntry }[] = [];
  for (let i = 0; i < history.length; i += 2) {
    rows.push({
      num: Math.floor(i / 2) + 1,
      white: history[i],
      black: history[i + 1],
    });
  }

  const navigate = (delta: number) => {
    if (delta === -Infinity) {
      onJump(0);
    } else if (delta === Infinity) {
      onJump(history.length);
    } else {
      onJump(Math.max(0, Math.min(history.length, currentPly + delta)));
    }
  };

  const whiteLow = clocks.white < 30000;
  const blackLow = clocks.black < 30000;

  return (
    <div className={styles.sidebar}>
      {clockEnabled && (
        <div className={styles.clocksRow}>
          <div
            className={`${styles.clockCard} ${activeColor === "black" ? styles.clockActive : ""} ${flagFallen === "black" ? styles.clockFlagged : ""}`}
          >
            <span className={styles.clockDotBlack} />
            <span className={`${styles.clockTime} ${blackLow ? styles.clockLow : ""}`}>
              {formatClock(clocks.black)}
            </span>
          </div>
          <div
            className={`${styles.clockCard} ${activeColor === "white" ? styles.clockActive : ""} ${flagFallen === "white" ? styles.clockFlagged : ""}`}
          >
            <span className={styles.clockDotWhite} />
            <span className={`${styles.clockTime} ${whiteLow ? styles.clockLow : ""}`}>
              {formatClock(clocks.white)}
            </span>
          </div>
        </div>
      )}

      <div className={styles.historyHeader}>
        <span className={styles.historyTitle}>Lances</span>
        <div className={styles.navButtons}>
          {NAV_BUTTONS.map((b) => (
            <button
              key={b.label}
              className={styles.navButton}
              title={b.title}
              onClick={() => navigate(b.delta)}
            >
              {b.label}
            </button>
          ))}
        </div>
      </div>

      <div className={styles.moveList} ref={listRef}>
        <button
          data-ply={0}
          className={`${styles.moveCell} ${styles.startCell} ${currentPly === 0 ? styles.startCellActive : ""}`}
          onClick={() => onJump(0)}
        >
          Início
        </button>
        {rows.map((row) => {
          const whitePly = (row.num - 1) * 2 + 1;
          const blackPly = whitePly + 1;
          return (
            <div key={row.num} className={styles.moveRow}>
              <span className={styles.moveNum}>{row.num}.</span>
              <button
                data-ply={whitePly}
                className={`${styles.moveCell} ${currentPly === whitePly ? styles.moveCellActive : ""}`}
                onClick={() => onJump(whitePly)}
              >
                {row.white && renderMoveText(row.white.san, "white")}
                {row.white?.analysis && (
                  <span
                    className={`${styles.evalTag} ${row.white.analysis.score < 0 ? styles.evalTagNegative : ""}`}
                  >
                    {formatEval(row.white.analysis.score)}
                  </span>
                )}
              </button>
              <button
                data-ply={blackPly}
                className={`${styles.moveCell} ${currentPly === blackPly ? styles.moveCellActive : ""}`}
                onClick={() => onJump(blackPly)}
                disabled={!row.black}
              >
                {row.black && renderMoveText(row.black.san, "black")}
                {row.black?.analysis && (
                  <span
                    className={`${styles.evalTag} ${row.black.analysis.score < 0 ? styles.evalTagNegative : ""}`}
                  >
                    {formatEval(row.black.analysis.score)}
                  </span>
                )}
              </button>
            </div>
          );
        })}
        {history.length === 0 && (
          <div className={styles.emptyHint}>Nenhum lance ainda</div>
        )}
      </div>

      {result && resultText && (
        <div className={styles.resultBox}>
          <span className={styles.resultText}>{resultText}</span>
          <button className={styles.restartButton} onClick={onRestart}>
            Jogar novamente
          </button>
        </div>
      )}

      {clockEnabled && flagFallen && !result && (
        <div className={styles.resultBox}>
          <span className={styles.resultText}>
            {flagFallen === "white" ? "Tempo das Brancas esgotado" : "Tempo das Pretas esgotado"}
          </span>
        </div>
      )}
    </div>
  );
};