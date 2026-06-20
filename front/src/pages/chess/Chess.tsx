import { useEffect, useState } from "react";
import { ChessBoard } from "@/components/ChessBoard/ChessBoard";
import { MoveHistory } from "@/components/MoveHistory/MoveHistory";
import { PromotionPicker } from "@/components/PromotionPicker/PromotionPicker";
import type { ClockConfig } from "@/hooks/useChessClock";
import { minutesToMs } from "@/hooks/useChessClock";
import type { AiAnalysisResult } from "@/wasm/generated/wasm-contract";
import type { ChessColor } from "@/types/chess";
import { squareName } from "@/utils/chessNotation";
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
  { value: "ai-vs-ai", label: "IA vs IA" },
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

const CLOCK_PRESETS = [
  { label: "Sem relógio", minutes: 0 },
  { label: "1 min", minutes: 1 },
  { label: "3 min", minutes: 3 },
  { label: "5 min", minutes: 5 },
  { label: "10 min", minutes: 10 },
  { label: "15 min", minutes: 15 },
];

const INCREMENT_PRESETS = [0, 2, 3, 5, 10];

export const Chess = () => {
  const [mode, setMode] = useState<ChessGameMode>("human-vs-human");
  const [aiColor, setAiColor] = useState<ChessColor>("black");
  const [difficulty, setDifficulty] = useState<ChessDifficulty>("medium");
  const [searchMode, setSearchMode] = useState<ChessSearchMode>("difficulty");
  const [customTimeMs, setCustomTimeMs] = useState(1000);
  const [customDepth, setCustomDepth] = useState(4);
  const [flipped, setFlipped] = useState(false);

  const [clockMinutes, setClockMinutes] = useState(0);
  const [clockIncrement, setClockIncrement] = useState(3);

  const clockConfig: ClockConfig = {
    enabled: clockMinutes > 0,
    initialMs: minutesToMs(clockMinutes),
    incrementMs: clockIncrement * 1000,
  };

  const {
    state,
    history,
    currentPly,
    isAtLatest,
    handleSquareClick,
    restartGame,
    choosePromotion,
    cancelPromotion,
    jumpToPly,
    clock,
    gameStarted,
    analyze,
    autoAnalyze,
    setAutoAnalyze,
  } = useChess(
    mode,
    aiColor,
    difficulty,
    searchMode,
    customTimeMs,
    customDepth,
    clockConfig,
  );

  const {
    board,
    currentPlayer,
    selectedSquare,
    validMoveSquares,
    result,
    pendingPromotion,
    checkSquare,
    aiThinking,
    lastMove,
    boardBefore,
    animateId,
  } = state;

  const resultText = result ? RESULT_TEXT[result] : null;
  const gameOver = result !== null && isAtLatest;

  const [analysis, setAnalysis] = useState<AiAnalysisResult | null>(null);
  const [analyzing, setAnalyzing] = useState(false);
  const [arrow, setArrow] = useState<{ from: number; to: number } | null>(null);

  const handleRestart = () => {
    setAnalysis(null);
    setArrow(null);
    restartGame();
  };

  const handleAnalyze = async () => {
    if (analyzing || result) return;
    setAnalyzing(true);
    setArrow(null);
    try {
      const res = await analyze(1000);
      setAnalysis(res);
      if (res) {
        setArrow({ from: res.from, to: res.to });
      }
    } finally {
      setAnalyzing(false);
    }
  };

  const handleCloseAnalysis = () => {
    setAnalysis(null);
    setArrow(null);
  };

  const handleClockPreset = (minutes: number) => {
    setClockMinutes(minutes);
    if (minutes > 0 && !gameStarted) {
      clock.reset({
        enabled: true,
        initialMs: minutesToMs(minutes),
        incrementMs: clockIncrement * 1000,
      });
    } else if (minutes === 0 && !gameStarted) {
      clock.reset({
        enabled: false,
        initialMs: 0,
        incrementMs: 0,
      });
    }
  };

  // ── Keyboard navigation for move history ────────────────
  useEffect(() => {
    if (pendingPromotion) return;

    const handler = (e: KeyboardEvent) => {
      if (e.target instanceof HTMLInputElement) return;

      switch (e.key) {
        case "ArrowLeft":
          e.preventDefault();
          setArrow(null);
          jumpToPly(currentPly - 1);
          break;
        case "ArrowRight":
          e.preventDefault();
          setArrow(null);
          jumpToPly(currentPly + 1);
          break;
        case "Home":
          e.preventDefault();
          setArrow(null);
          jumpToPly(0);
          break;
        case "End":
          e.preventDefault();
          setArrow(null);
          jumpToPly(history.length);
          break;
      }
    };

    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [currentPly, history.length, jumpToPly, pendingPromotion]);

  return (
    <div className={styles.page}>
      <div className={styles.layout}>
        {/* ── Left: board area ─────────────────────────── */}
        <div className={styles.boardArea}>
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
                {COLORS.map((c) => {
                  const humanIsThisColor = aiColor !== c.value;
                  return (
                    <button
                      key={c.value}
                      className={`${styles.modeButton} ${humanIsThisColor ? styles.modeButtonActive : ""}`}
                      onClick={() => {
                        setAiColor(c.value === "white" ? "black" : "white");
                        setFlipped(c.value === "black");
                        restartGame();
                      }}
                    >
                      {c.label}
                    </button>
                  );
                })}
              </div>
            </div>
          )}

          {(mode === "human-vs-ai" || mode === "ai-vs-ai") && (
            <div className={styles.aiSetup}>
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
                    onChange={(e) =>
                      setCustomTimeMs(Math.max(10, Number(e.target.value) || 1000))
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
                      setCustomDepth(
                        Math.max(1, Math.min(10, Number(e.target.value) || 4)),
                      )
                    }
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
              {gameOver && resultText ? (
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

          <ChessBoard
            board={board}
            selectedSquare={selectedSquare}
            validMoveSquares={validMoveSquares}
            onSquareClick={handleSquareClick}
            flipped={flipped}
            checkSquare={checkSquare}
            lastMove={lastMove}
            boardBefore={boardBefore}
            animateId={animateId}
            arrow={arrow}
          />

          <div className={styles.actions}>
            <button className={styles.actionButton} onClick={handleRestart}>
              Reiniciar
            </button>
            <button
              className={`${styles.actionButton} ${flipped ? styles.actionButtonActive : ""}`}
              onClick={() => setFlipped((f) => !f)}
            >
              Girar ↺
            </button>
            {!result && (
              <button
                className={styles.actionButton}
                onClick={handleAnalyze}
                disabled={analyzing || aiThinking}
              >
                {analyzing ? "Analisando..." : "Analisar"}
              </button>
            )}
            {!result && (
              <button
                className={`${styles.actionButton} ${autoAnalyze ? styles.actionButtonActive : ""}`}
                onClick={() => setAutoAnalyze(!autoAnalyze)}
                title="Analisa automaticamente cada posição após cada lance"
              >
                {autoAnalyze ? "Auto ✓" : "Analisar auto"}
              </button>
            )}
          </div>

          {analysis && (
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
              <button
                className={styles.closeAnalysis}
                onClick={handleCloseAnalysis}
              >
                ✕
              </button>
            </div>
          )}
        </div>

        {/* ── Right: sidebar ────────────────────────────── */}
        <div className={styles.sidebarArea}>
          <div className={styles.clockConfig}>
            <span className={styles.configTitle}>Relógio</span>
            <div className={styles.presetRow}>
              {CLOCK_PRESETS.map((p) => (
                <button
                  key={p.label}
                  className={`${styles.presetButton} ${clockMinutes === p.minutes ? styles.presetButtonActive : ""}`}
                  onClick={() => handleClockPreset(p.minutes)}
                  disabled={gameStarted && clockConfig.enabled}
                >
                  {p.label}
                </button>
              ))}
            </div>
            {clockConfig.enabled && (
              <div className={styles.presetRow}>
                <span className={styles.incrementLabel}>Incremento:</span>
                {INCREMENT_PRESETS.map((s) => (
                  <button
                    key={s}
                    className={`${styles.presetButtonSmall} ${clockIncrement === s ? styles.presetButtonActive : ""}`}
                    onClick={() => setClockIncrement(s)}
                    disabled={gameStarted}
                  >
                    {s}s
                  </button>
                ))}
              </div>
            )}
          </div>

          <MoveHistory
            history={history}
            currentPly={currentPly}
            onJump={jumpToPly}
            clocks={clock.clocks}
            activeColor={currentPlayer}
            clockEnabled={clockConfig.enabled}
            flagFallen={clock.flagFallen}
            result={result}
            resultText={resultText}
            onRestart={handleRestart}
          />
        </div>
      </div>


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