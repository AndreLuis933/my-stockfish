import { useState } from "react";
import type { GameMode } from "@/hooks/useGame";
import { useGame } from "@/hooks/useGame";
import { Board } from "@/components/Board/Board";
import styles from "./App.module.css";

const RESULT_TEXT = {
  "white-wins": "Brancas vencem!",
  "black-wins": "Pretas vencem!",
  draw: "Empate!",
};

const MODES: { value: GameMode; label: string }[] = [
  { value: "human-vs-ai", label: "Humano vs IA" },
  { value: "human-vs-human", label: "Humano vs Humano" },
  { value: "ai-vs-ai", label: "IA vs IA" },
];

function App() {
  const [mode, setMode] = useState<GameMode>("human-vs-ai");
  const { state, handleSquareClick, restartGame } = useGame(mode);
  const { board, currentPlayer, selectedSquare, movesForSelected, turnState, flashSelectable, result } = state;

  const aiThinking = mode === "ai-vs-ai" || (mode === "human-vs-ai" && currentPlayer === "black");

  return (
    <div className={styles.page}>
      <div className={styles.modeSelector}>
        {MODES.map((m) => (
          <button
            key={m.value}
            className={`${styles.modeButton} ${mode === m.value ? styles.modeButtonActive : ""}`}
            onClick={() => setMode(m.value)}
          >
            {m.label}
          </button>
        ))}
      </div>

      <div className={styles.turnBanner}>
        <div className={`${styles.dot} ${styles.dotWhite} ${currentPlayer === "white" ? styles.active : ""}`} />
        <span className={styles.turnText}>
          {aiThinking && result === null
            ? "IA pensando..."
            : currentPlayer === "white"
              ? "Vez das Brancas"
              : "Vez das Pretas"}
        </span>
        <div className={`${styles.dot} ${styles.dotBlack} ${currentPlayer === "black" ? styles.active : ""}`} />
      </div>

      <Board
        board={board}
        selectedSquare={selectedSquare}
        validMoveSquares={movesForSelected.map((m) => m.to)}
        mustMoveSquares={flashSelectable ? turnState.selectable : []}
        onSquareClick={handleSquareClick}
      />

      {result && (
        <div className={styles.overlay}>
          <div className={styles.resultCard}>
            <span className={styles.resultTitle}>{RESULT_TEXT[result]}</span>
            <button className={styles.restartButton} onClick={restartGame}>
              Jogar novamente
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

export default App;
