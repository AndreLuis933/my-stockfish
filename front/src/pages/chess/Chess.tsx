import { useState } from "react";
import { ChessBoard } from "@/components/ChessBoard/ChessBoard";
import { PromotionPicker } from "@/components/PromotionPicker/PromotionPicker";
import type { ChessGameMode } from "./Chess.hooks";
import { useChess } from "./Chess.hooks";
import styles from "./Chess.module.css";

const RESULT_TEXT = {
  "white-wins": "Brancas vencem!",
  "black-wins": "Pretas vencem!",
  draw: "Empate!",
};

const MODES: { value: ChessGameMode; label: string }[] = [
  { value: "human-vs-ai", label: "Humano vs IA" },
  { value: "human-vs-human", label: "Humano vs Humano" },
];

export const Chess = () => {
  const [mode, setMode] = useState<ChessGameMode>("human-vs-human");
  const [flipped, setFlipped] = useState(false);
  const { state, handleSquareClick, restartGame, choosePromotion, cancelPromotion } = useChess(mode);
  const { board, currentPlayer, selectedSquare, validMoveSquares, result, pendingPromotion } = state;
  

  return (
    <div className={styles.page}>
      <div className={styles.modeSelector}>
        {MODES.map((m) => (
          <button
            key={m.value}
            className={`${styles.modeButton} ${mode === m.value ? styles.modeButtonActive : ""}`}
            onClick={() => {
              restartGame();
              setMode(m.value);
            }}
          >
            {m.label}
          </button>
        ))}
      </div>

      <div className={styles.turnBanner}>
        <div
          className={`${styles.dot} ${styles.dotWhite} ${currentPlayer === "white" ? styles.active : ""}`}
        />
        <span className={styles.turnText}>
          {currentPlayer === "white" ? "Vez das Brancas" : "Vez das Pretas"}
        </span>
        <div
          className={`${styles.dot} ${styles.dotBlack} ${currentPlayer === "black" ? styles.active : ""}`}
        />
      </div>

      <ChessBoard
        board={board}
        selectedSquare={selectedSquare}
        validMoveSquares={validMoveSquares}
        onSquareClick={handleSquareClick}
        flipped={flipped}
      />

      <div className={styles.engineNotice}>
        Engine Go em desenvolvimento — movimentos ainda não validados
      </div>

      <div className={styles.actions}>
        <button className={styles.actionButton} onClick={restartGame}>
          Reiniciar
        </button>
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

      {pendingPromotion && (
        <PromotionPicker
          options={pendingPromotion.options}
          onSelect={choosePromotion}
          onCancel={cancelPromotion}
        />
      )}
    </div>
  );
};
