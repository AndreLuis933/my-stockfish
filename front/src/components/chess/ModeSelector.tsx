import type { ChessGameMode } from "@/pages/chess/Chess.types";
import styles from "./ChessShared.module.css";

const MODES: { value: ChessGameMode; label: string }[] = [
  { value: "human-vs-ai", label: "Humano vs IA" },
  { value: "human-vs-human", label: "Humano vs Humano" },
  { value: "ai-vs-ai", label: "IA vs IA" },
  { value: "analysis", label: "Análise" },
];

interface ModeSelectorProps {
  mode: ChessGameMode;
  onModeChange: (mode: ChessGameMode) => void;
}

export const ModeSelector = ({ mode, onModeChange }: ModeSelectorProps) => (
  <div className={styles.modeSelector}>
    {MODES.map((m) => (
      <button
        key={m.value}
        className={`${styles.modeButton} ${mode === m.value ? styles.modeButtonActive : ""}`}
        onClick={() => onModeChange(m.value)}
      >
        {m.label}
      </button>
    ))}
  </div>
);