import { useState } from "react";
import { ChessBoard } from "@/components/ChessBoard/ChessBoard";
import { PromotionPicker } from "@/components/PromotionPicker/PromotionPicker";
import type { ChessColor } from "@/types/chess";
import type { ChessDifficulty, ChessGameMode, ChessSearchMode } from "./Chess.hooks";
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

const DIFFICULTIES: { value: ChessDifficulty; label: string }[] = [
  { value: "easy", label: "Fácil" },
  { value: "medium", label: "Médio" },
  { value: "hard", label: "Difícil" },
];

const COLORS: { value: ChessColor; label: string }[] = [
  { value: "white", label: "Brancas" },
  { value: "black", label: "Pretas" },
];

const SEARCH_MODES: { value: ChessSearchMode; label: string }[] = [
  { value: "difficulty", label: "Dificuldade" },
  { value: "time", label: "Tempo (ms)" },
  { value: "depth", label: "Profundidade" },
];

export const Chess = () => {
  const [mode, setMode] = useState<ChessGameMode>("human-vs-human");
  const [aiColor, setAiColor] = useState<ChessColor>("black");
  const [difficulty, setDifficulty] = useState<ChessDifficulty>("medium");
  const [searchMode, setSearchMode] = useState<ChessSearchMode>("difficulty");
  const [customTimeMs, setCustomTimeMs] = useState(1000);
  const [customDepth, setCustomDepth] = useState(4);
  const [flipped, setFlipped] = useState(false);
  const { state, handleSquareClick, restartGame, choosePromotion, cancelPromotion } =
    useChess(mode, aiColor, difficulty, searchMode, customTimeMs, customDepth);
  const {
    board,
    currentPlayer,
    selectedSquare,
    validMoveSquares,
    result,
    pendingPromotion,
    checkSquare,
    aiThinking,
  } = state;

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

      {mode === "human-vs-ai" && (
        <div className={styles.aiSetup}>
          <div className={styles.aiSetupGroup}>
            <span className={styles.aiSetupLabel}>Você joga de:</span>
            {COLORS.map((c) => (
              <button
                key={c.value}
                className={`${styles.modeButton} ${aiColor === c.value ? styles.modeButtonActive : ""}`}
                onClick={() => {
                  const humanColor: ChessColor = c.value === "white" ? "black" : "white";
                  setAiColor(humanColor);
                  setFlipped(c.value === "black");
                  restartGame();
                }}
              >
                {c.label}
              </button>
            ))}
          </div>

          <div className={styles.aiSetupGroup}>
            <span className={styles.aiSetupLabel}>Busca:</span>
            {SEARCH_MODES.map((s) => (
              <button
                key={s.value}
                className={`${styles.modeButton} ${searchMode === s.value ? styles.modeButtonActive : ""}`}
                onClick={() => setSearchMode(s.value)}
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
                  onClick={() => setDifficulty(d.value)}
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
                onChange={(e) => setCustomTimeMs(Math.max(10, Number(e.target.value) || 1000))}
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
                onChange={(e) => setCustomDepth(Math.max(1, Math.min(10, Number(e.target.value) || 4)))}
                className={styles.numberInput}
              />
            </div>
          )}
        </div>
      )}

      <div className={styles.turnBanner}>
        <div
          className={`${styles.dot} ${styles.dotWhite} ${currentPlayer === "white" ? styles.active : ""}`}
        />
        <span className={styles.turnText}>
          {currentPlayer === "white" ? "Vez das Brancas" : "Vez das Pretas"}
          {aiThinking && <span className={styles.thinkingBadge}>IA pensando...</span>}
          {checkSquare !== null && <span className={styles.checkBadge}>Xeque!</span>}
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
        checkSquare={checkSquare}
      />

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