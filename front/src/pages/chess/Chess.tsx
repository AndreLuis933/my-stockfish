import { useEffect, useState } from "react";
import { ChessBoard } from "@/components/ChessBoard/ChessBoard";
import { MoveHistory } from "@/components/MoveHistory/MoveHistory";
import { PromotionPicker } from "@/components/PromotionPicker/PromotionPicker";
import { AnalysisPanel } from "@/components/AnalysisPanel/AnalysisPanel";
import { BottomBar } from "@/components/BottomBar/BottomBar";
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
  { value: "analysis", label: "Análise" },
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
    analyzeCurrentPosition,
    analysisForPly,
    exportPgn,
    loadPgn,
    multiPv,
    multiPvSan,
    multiPvDepth,
    multiPvThinking,
    runMultiPv,
    stopMultiPv,
    currentFen,
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

  const isAnalysisMode = mode === "analysis";

  const [analysis, setAnalysis] = useState<AiAnalysisResult | null>(null);
  const [analyzing, setAnalyzing] = useState(false);
  const [analysisEnabled, setAnalysisEnabled] = useState(false);
  const [maxLines, setMaxLines] = useState(1);
  const [analysisTimeMs, setAnalysisTimeMs] = useState(1000);

  const [pgnModalOpen, setPgnModalOpen] = useState(false);
  const [pgnText, setPgnText] = useState("");
  const [pgnError, setPgnError] = useState<string | null>(null);
  const [copyStatus, setCopyStatus] = useState<"idle" | "copied">("idle");

  const savedAnalysis = isAtLatest ? null : analysisForPly(currentPly) ?? null;
  const displayedAnalysis = isAtLatest ? analysis : savedAnalysis;
  const arrow = displayedAnalysis
    ? { from: displayedAnalysis.from, to: displayedAnalysis.to }
    : null;

  const handleRestart = () => {
    setAnalysis(null);
    setPgnText("");
    setPgnError(null);
    setAnalysisEnabled(false);
    restartGame();
  };

  const handleAnalyze = async () => {
    if (analyzing) return;
    if (result && isAtLatest) return;
    setAnalyzing(true);
    try {
      if (isAnalysisMode || !isAtLatest) {
        await analyzeCurrentPosition();
      } else {
        const res = await analyze(1000);
        setAnalysis(res);
      }
    } finally {
      setAnalyzing(false);
    }
  };

  const handleCloseAnalysis = () => {
    setAnalysis(null);
  };

  const handleCopyPgn = async () => {
    const pgn = exportPgn();
    if (!pgn) return;
    try {
      await navigator.clipboard.writeText(pgn);
      setCopyStatus("copied");
      setTimeout(() => setCopyStatus("idle"), 2000);
    } catch {
      setPgnModalOpen(true);
      setPgnText(pgn);
    }
  };

  const handleLoadPgn = async () => {
    setPgnError(null);
    if (!pgnText.trim()) {
      setPgnError("Cole um PGN primeiro.");
      return;
    }
    const ok = await loadPgn(pgnText);
    if (!ok) {
      setPgnError("Não foi possível ler o PGN. Verifique a notação.");
      return;
    }
    setPgnModalOpen(false);
    setAnalysis(null);
  };

  const handleModeChange = (newMode: ChessGameMode) => {
    handleRestart();
    setMode(newMode);
    if (newMode === "analysis") {
      setAutoAnalyze(true);
    }
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
          jumpToPly(currentPly - 1);
          break;
        case "ArrowRight":
          e.preventDefault();
          jumpToPly(currentPly + 1);
          break;
        case "Home":
          e.preventDefault();
          jumpToPly(0);
          break;
        case "End":
          e.preventDefault();
          jumpToPly(history.length);
          break;
      }
    };

    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [currentPly, history.length, jumpToPly, pendingPromotion]);

  // ── Continuous Multi-PV analysis (analysis mode, user-toggled) ───
  useEffect(() => {
    if (!isAnalysisMode || !analysisEnabled) {
      stopMultiPv();
      return;
    }
    if (result && isAtLatest) {
      stopMultiPv();
      return;
    }
    runMultiPv(maxLines, analysisTimeMs);
    return () => stopMultiPv();
  }, [isAnalysisMode, analysisEnabled, maxLines, analysisTimeMs, result, isAtLatest, currentPly, runMultiPv, stopMultiPv]);

  const handleToggleAnalysis = () => {
    setAnalysisEnabled((v) => !v);
  };

  const handleMaxLinesChange = (n: number) => {
    setMaxLines(n);
  };

  const handleAnalysisTimeChange = (ms: number) => {
    setAnalysisTimeMs(ms);
  };

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
                onClick={() => handleModeChange(m.value)}
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

          {isAnalysisMode && (
            <BottomBar
              fen={currentFen}
              pgn={exportPgn()}
              onJumpToStart={() => jumpToPly(0)}
              onJumpToEnd={() => jumpToPly(history.length)}
              onStepBack={() => jumpToPly(currentPly - 1)}
              onStepForward={() => jumpToPly(currentPly + 1)}
              canStepBack={currentPly > 0}
              canStepForward={currentPly < history.length}
            />
          )}

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
            {history.length > 0 && (
              <button
                className={`${styles.actionButton} ${copyStatus === "copied" ? styles.actionButtonActive : ""}`}
                onClick={handleCopyPgn}
                title="Copia o PGN da partida para a área de transferência"
              >
                {copyStatus === "copied" ? "Copiado ✓" : "Copiar PGN"}
              </button>
            )}
            <button
              className={styles.actionButton}
              onClick={() => {
                setPgnText("");
                setPgnError(null);
                setPgnModalOpen(true);
              }}
              title="Cola um PGN para visualizar a partida"
            >
              Colar PGN
            </button>
            {(!result || !isAtLatest) && (
              <button
                className={styles.actionButton}
                onClick={handleAnalyze}
                disabled={analyzing || aiThinking}
              >
                {analyzing ? "Analisando..." : "Analisar"}
              </button>
            )}
            {(!result || isAnalysisMode) && (
              <button
                className={`${styles.actionButton} ${autoAnalyze ? styles.actionButtonActive : ""}`}
                onClick={() => setAutoAnalyze(!autoAnalyze)}
                title="Analisa automaticamente cada posição após cada lance"
              >
                {autoAnalyze ? "Auto ✓" : "Analisar auto"}
              </button>
            )}
            {isAnalysisMode && (
              <button
                className={`${styles.actionButton} ${analysisEnabled ? styles.actionButtonActive : ""}`}
                onClick={handleToggleAnalysis}
                title={analysisEnabled ? "Parar análise contínua" : "Iniciar análise contínua"}
              >
                {analysisEnabled ? "Análise ⏸" : "Análise ▶"}
              </button>
            )}
          </div>

          {displayedAnalysis && (
            <div className={styles.analysisPanel}>
              <div className={styles.analysisRow}>
                <span className={styles.analysisLabel}>Avaliação</span>
                <span className={styles.analysisValue}>
                  {(displayedAnalysis.score / 100).toFixed(2)}
                </span>
              </div>
              <div className={styles.analysisRow}>
                <span className={styles.analysisLabel}>Melhor lance</span>
                <span className={styles.analysisValue}>
                  {squareName(displayedAnalysis.from)}→{squareName(displayedAnalysis.to)}
                </span>
              </div>
              <div className={styles.analysisRow}>
                <span className={styles.analysisLabel}>Profundidade</span>
                <span className={styles.analysisValue}>{displayedAnalysis.depth}</span>
              </div>
              {isAtLatest && (
                <button
                  className={styles.closeAnalysis}
                  onClick={handleCloseAnalysis}
                >
                  ✕
                </button>
              )}
            </div>
          )}
        </div>

        {/* ── Right: sidebar ────────────────────────────── */}
        <div className={styles.sidebarArea}>
          {isAnalysisMode ? (
            <AnalysisPanel
              lines={multiPv}
              thinking={multiPvThinking}
              depth={multiPvDepth}
              pvSanLines={multiPvSan}
              blunderPly={null}
              analysisEnabled={analysisEnabled}
              onToggleAnalysis={handleToggleAnalysis}
              maxLines={maxLines}
              onMaxLinesChange={handleMaxLinesChange}
              analysisTimeMs={analysisTimeMs}
              onAnalysisTimeChange={handleAnalysisTimeChange}
            />
          ) : (
            <>
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
            </>
          )}
        </div>
      </div>


      {pendingPromotion && (
        <PromotionPicker
          options={pendingPromotion.options}
          onSelect={choosePromotion}
          onCancel={cancelPromotion}
        />
      )}

      {pgnModalOpen && (
        <div className={styles.pgnOverlay} onClick={() => setPgnModalOpen(false)}>
          <div className={styles.pgnModal} onClick={(e) => e.stopPropagation()}>
            <div className={styles.pgnModalHeader}>
              <span className={styles.pgnModalTitle}>Importar PGN</span>
              <button
                className={styles.pgnCloseBtn}
                onClick={() => setPgnModalOpen(false)}
              >
                ✕
              </button>
            </div>
            <p className={styles.pgnHint}>
              Cole a notação PGN de uma partida para visualizá-la lance a lance.
            </p>
            <textarea
              className={styles.pgnTextarea}
              value={pgnText}
              onChange={(e) => {
                setPgnText(e.target.value);
                setPgnError(null);
              }}
              placeholder="[Event &quot;...&quot;]&#10;1. e4 e5 2. Nf3 Nc6 ..."
              rows={10}
              autoFocus
            />
            {pgnError && <div className={styles.pgnError}>{pgnError}</div>}
            <div className={styles.pgnModalActions}>
              <button
                className={styles.actionButton}
                onClick={() => setPgnModalOpen(false)}
              >
                Cancelar
              </button>
              <button
                className={styles.pgnLoadButton}
                onClick={handleLoadPgn}
                disabled={!pgnText.trim()}
              >
                Carregar
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};