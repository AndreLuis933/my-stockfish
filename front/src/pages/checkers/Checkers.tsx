import { useState } from "react";
import type { GameMode } from "@/hooks/useGame";
import { useGame } from "@/hooks/useGame";
import { Board } from "@/components/Board/Board";
import styles from "./Checkers.module.css";

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

export const Checkers = () => {
  const [mode, setMode] = useState<GameMode>("human-vs-ai");
  const [flipped, setFlipped] = useState(false);
  const { state, handleSquareClick, restartGame } = useGame(mode);
  const { board, currentPlayer, selectedSquare, movesForSelected, turnState, flashSelectable, result } = state;

  const aiThinking = mode === "ai-vs-ai" || (mode === "human-vs-ai" && currentPlayer === "black");

  const pieceCounts = board.reduce(
    (acc, row) => {
      row.forEach((cell) => { if (cell) acc[cell.color]++; });
      return acc;
    },
    { white: 0, black: 0 }
  );

  return (
    <div className={styles.page}>
      <div className={styles.modeSelector}>
        {MODES.map((m) => (
          <button
            key={m.value}
            className={`${styles.modeButton} ${mode === m.value ? styles.modeButtonActive : ""}`}
            onClick={() => { restartGame(); setMode(m.value); }}
          >
            {m.label}
          </button>
        ))}
      </div>

      <div className={styles.turnBanner}>
        <div className={styles.sideInfo}>
          <div className={`${styles.dot} ${styles.dotWhite} ${currentPlayer === "white" ? styles.active : ""}`} />
          <span className={styles.pieceCountText}>{pieceCounts.white}</span>
        </div>
        <span className={styles.turnText}>
          {aiThinking && result === null
            ? "IA pensando..."
            : currentPlayer === "white"
              ? "Vez das Brancas"
              : "Vez das Pretas"}
        </span>
        <div className={styles.sideInfo}>
          <span className={styles.pieceCountText}>{pieceCounts.black}</span>
          <div className={`${styles.dot} ${styles.dotBlack} ${currentPlayer === "black" ? styles.active : ""}`} />
        </div>
      </div>

      <Board
        board={board}
        selectedSquare={selectedSquare}
        validMoveSquares={movesForSelected.map((m) => m.to)}
        mustMoveSquares={flashSelectable ? turnState.selectable : []}
        onSquareClick={handleSquareClick}
        flipped={flipped}
      />

      <div className={styles.actions}>
        <button className={styles.actionButton} onClick={restartGame}>Reiniciar</button>
        <button
          className={`${styles.actionButton} ${flipped ? styles.actionButtonActive : ""}`}
          onClick={() => setFlipped((f) => !f)}
        >
          Girar ↺
        </button>
      </div>

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
};
