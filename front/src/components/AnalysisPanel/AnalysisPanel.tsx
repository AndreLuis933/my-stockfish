import { useState } from "react";
import { Figurine } from "@/components/Figurine/Figurine";
import type { MultiPvLine } from "@/wasm/generated/wasm-contract";
import type { ChessColor } from "@/types/chess";
import { formatEval } from "@/utils/chessAnalysis";
import { parseSanFigurine } from "@/utils/chessFigurine";
import type { PvSanEntry } from "@/utils/pvToSan";
import styles from "./AnalysisPanel.module.css";

interface AnalysisPanelProps {
  lines: MultiPvLine[];
  thinking: boolean;
  depth: number;
  pvSanLines: PvSanEntry[][];
  blunderPly: number | null;
  analysisEnabled: boolean;
  onToggleAnalysis: () => void;
  maxLines: number;
  onMaxLinesChange: (n: number) => void;
  analysisTimeMs: number;
  onAnalysisTimeChange: (ms: number) => void;
}

const renderSanWithFigurine = (
  san: string,
  color: ChessColor,
  isBlunder: boolean,
  isMistake: boolean,
): React.ReactNode => {
  const { pieceType, rest } = parseSanFigurine(san);
  return (
    <span className={styles.pvMovePair}>
      {pieceType && (
        <Figurine
          type={pieceType}
          color={color}
          className={styles.pvFigurine}
        />
      )}
      <span
        className={
          isBlunder
            ? styles.blunderMove
            : isMistake
              ? styles.mistakeMove
              : undefined
        }
      >
        {rest}
      </span>
    </span>
  );
};

export const AnalysisPanel = ({
  lines,
  thinking,
  depth,
  pvSanLines,
  blunderPly,
  analysisEnabled,
  onToggleAnalysis,
  maxLines,
  onMaxLinesChange,
  analysisTimeMs,
  onAnalysisTimeChange,
}: AnalysisPanelProps) => {
  const [configOpen, setConfigOpen] = useState(false);
  const topLine = lines[0];
  const topScore = topLine?.score ?? 0;

  const mainLinePreview = pvSanLines[0]
    ? pvSanLines[0]
        .map((e, i) => {
          const num = Math.floor(i / 2) + 1;
          return i % 2 === 0 ? `${num}.${e.san}` : e.san;
        })
        .join(" ")
    : "";

  return (
    <div className={styles.panel}>
      <div className={styles.header}>
        <div className={styles.headerTop}>
          <span
            className={`${styles.statusDot} ${analysisEnabled && !thinking ? "" : styles.statusDotIdle}`}
            onClick={onToggleAnalysis}
            title={analysisEnabled ? "Parar análise" : "Iniciar análise"}
            style={{ cursor: "pointer" }}
          />
          <span className={styles.evalScore}>{formatEval(topScore)}</span>
          <span className={styles.headerIcons}>
            <span
              className={`${styles.headerIcon} ${analysisEnabled ? styles.headerIconActive : ""}`}
              onClick={onToggleAnalysis}
              title={analysisEnabled ? "Parar análise" : "Iniciar análise"}
            >
              {analysisEnabled ? "⏸" : "▶"}
            </span>
            <span
              className={styles.headerIcon}
              title="Configurações"
              onClick={() => setConfigOpen((v) => !v)}
            >
              ⚙
            </span>
          </span>
        </div>

        {configOpen && (
          <div className={styles.configPanel}>
            <div className={styles.configRow}>
              <span className={styles.configLabel}>Linhas Multi-PV:</span>
              <div className={styles.configButtons}>
                {[1, 2, 3, 4, 5].map((n) => (
                  <button
                    key={n}
                    className={`${styles.configBtn} ${maxLines === n ? styles.configBtnActive : ""}`}
                    onClick={() => onMaxLinesChange(n)}
                  >
                    {n}
                  </button>
                ))}
              </div>
            </div>
            <div className={styles.configRow}>
              <span className={styles.configLabel}>Tempo por lance:</span>
              <div className={styles.configButtons}>
                {[200, 500, 1000, 2000, 5000].map((ms) => (
                  <button
                    key={ms}
                    className={`${styles.configBtn} ${analysisTimeMs === ms ? styles.configBtnActive : ""}`}
                    onClick={() => onAnalysisTimeChange(ms)}
                  >
                    {ms >= 1000 ? `${ms / 1000}s` : `${ms}ms`}
                  </button>
                ))}
              </div>
            </div>
            <button
              className={styles.configClose}
              onClick={() => setConfigOpen(false)}
            >
              Fechar
            </button>
          </div>
        )}

        <div className={styles.engineInfo}>
          <span className={styles.engineFlag}>🚩</span>
          <span className={styles.engineName}>my-stockfish</span>{" "}
          <span className={styles.engineBadge}>Negamax</span>
        </div>
        <div className={styles.depthLine}>
          <span className={styles.engineFlag}>🚩</span>
          <span>
            Profundidade {depth || (topLine?.depth ?? 0)}
            {maxLines > 1 && ` · ${maxLines} linhas`}
            {` · ${analysisTimeMs >= 1000 ? `${analysisTimeMs / 1000}s` : `${analysisTimeMs}ms`}`}
          </span>
        </div>
      </div>

      {mainLinePreview && (
        <div className={styles.mainLineRow} title={mainLinePreview}>
          <span className={styles.mainLineText}>{mainLinePreview}</span>
          <span className={styles.expandCaret}>▼</span>
        </div>
      )}

      {lines.length === 0 && !thinking && (
        <div className={styles.empty}>Sem análise ainda</div>
      )}

      {thinking && lines.length === 0 && (
        <div className={styles.loading}>
          <span className={styles.spinner} />
          Calculando...
        </div>
      )}

      {lines.length > 0 && (
        <div className={styles.pvList}>
          {lines.map((line, lineIdx) => {
            const sanEntries = pvSanLines[lineIdx] ?? [];
            const isHighlighted = blunderPly !== null && lineIdx === 0;
            const scoreClass =
              line.score < 0 ? styles.pvEvalNegative : styles.pvEval;

            return (
              <div key={lineIdx}>
                <div
                  className={`${styles.pvRow} ${isHighlighted ? styles.pvRowHighlight : ""}`}
                >
                  <span className={styles.pvRank}>{lineIdx + 1}</span>
                  <span className={styles.pvMoves}>
                    {sanEntries.map((entry, i) => {
                      const moveNum = Math.floor(i / 2) + 1;
                      const showNum = i % 2 === 0;
                      return (
                        <span key={i} className={styles.pvMovePair}>
                          {showNum && (
                            <span className={styles.pvMoveNum}>{moveNum}.</span>
                          )}
                          {renderSanWithFigurine(
                            entry.san,
                            entry.color,
                            false,
                            false,
                          )}
                        </span>
                      );
                    })}
                  </span>
                  <span className={scoreClass}>{formatEval(line.score)}</span>
                  {isHighlighted && blunderPly !== null && (
                    <span className={styles.blunderBadge}>
                      {formatEval(line.score)}
                    </span>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
};