import type { ChessColor } from "@/types/chess";
import type { ChessDifficulty, ChessSearchMode } from "@/pages/chess/Chess.types";
import styles from "./ChessShared.module.css";

const COLORS: { value: ChessColor; label: string }[] = [
  { value: "white", label: "Brancas" },
  { value: "black", label: "Pretas" },
];

const DIFFICULTIES: { value: ChessDifficulty; label: string }[] = [
  { value: "easy", label: "Fácil" },
  { value: "medium", label: "Médio" },
  { value: "hard", label: "Difícil" },
];

const SEARCH_MODES: { value: ChessSearchMode; label: string }[] = [
  { value: "difficulty", label: "Dificuldade" },
  { value: "time", label: "Tempo (ms)" },
  { value: "depth", label: "Profundidade" },
];

interface AiSetupPanelProps {
  mode: "human-vs-ai" | "ai-vs-ai";
  aiColor: ChessColor;
  onAiColorChange: (color: ChessColor) => void;
  searchMode: ChessSearchMode;
  onSearchModeChange: (mode: ChessSearchMode) => void;
  difficulty: ChessDifficulty;
  onDifficultyChange: (d: ChessDifficulty) => void;
  customTimeMs: number;
  onCustomTimeMsChange: (ms: number) => void;
  customDepth: number;
  onCustomDepthChange: (depth: number) => void;
  onRestart: () => void;
  showColorChoice: boolean;
}

export const AiSetupPanel = ({
  aiColor,
  onAiColorChange,
  searchMode,
  onSearchModeChange,
  difficulty,
  onDifficultyChange,
  customTimeMs,
  onCustomTimeMsChange,
  customDepth,
  onCustomDepthChange,
  onRestart,
  showColorChoice,
}: AiSetupPanelProps) => (
  <div className={styles.aiSetup}>
    {showColorChoice && (
      <div className={styles.aiSetupGroup}>
        <span className={styles.aiSetupLabel}>Você joga de:</span>
        {COLORS.map((c) => {
          const humanIsThisColor = aiColor !== c.value;
          return (
            <button
              key={c.value}
              className={`${styles.modeButton} ${humanIsThisColor ? styles.modeButtonActive : ""}`}
              onClick={() => {
                onAiColorChange(c.value === "white" ? "black" : "white");
                onRestart();
              }}
            >
              {c.label}
            </button>
          );
        })}
      </div>
    )}

    <div className={styles.aiSetupGroup}>
      <span className={styles.aiSetupLabel}>Busca:</span>
      {SEARCH_MODES.map((s) => (
        <button
          key={s.value}
          className={`${styles.modeButton} ${searchMode === s.value ? styles.modeButtonActive : ""}`}
          onClick={() => onSearchModeChange(s.value)}
        >
          {s.label}
        </button>
      ))}
    </div>

    {searchMode === "difficulty" && (
      <div className={styles.aiSetupGroup}>
        <span className={styles.aiSetupLabel}>Dificuldade:</span>
        {DIFFICULTIES.map((d) => (
          <button
            key={d.value}
            className={`${styles.modeButton} ${difficulty === d.value ? styles.modeButtonActive : ""}`}
            onClick={() => onDifficultyChange(d.value)}
          >
            {d.label}
          </button>
        ))}
      </div>
    )}

    {searchMode === "time" && (
      <div className={styles.aiSetupGroup}>
        <span className={styles.aiSetupLabel}>Tempo (ms):</span>
        <input
          type="number"
          min={10}
          max={60000}
          step={100}
          value={customTimeMs}
          onChange={(e) =>
            onCustomTimeMsChange(Math.max(10, Number(e.target.value) || 1000))
          }
          className={styles.numberInput}
        />
      </div>
    )}

    {searchMode === "depth" && (
      <div className={styles.aiSetupGroup}>
        <span className={styles.aiSetupLabel}>Profundidade:</span>
        <input
          type="number"
          min={1}
          max={10}
          step={1}
          value={customDepth}
          onChange={(e) =>
            onCustomDepthChange(
              Math.max(1, Math.min(10, Number(e.target.value) || 4)),
            )
          }
          className={styles.numberInput}
        />
      </div>
    )}
  </div>
);