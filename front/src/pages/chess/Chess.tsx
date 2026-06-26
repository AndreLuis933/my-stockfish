import { useState } from "react";
import { ChessBoard } from "@/components/ChessBoard/ChessBoard";
import { MoveHistory } from "@/components/MoveHistory/MoveHistory";
import { PromotionPicker } from "@/components/PromotionPicker/PromotionPicker";
import { AnalysisPanel } from "@/components/AnalysisPanel/AnalysisPanel";
import { BottomBar } from "@/components/BottomBar/BottomBar";
import { ModeSelector } from "@/components/chess/ModeSelector";
import { AiSetupPanel } from "@/components/chess/AiSetupPanel";
import { TurnBanner } from "@/components/chess/TurnBanner";
import { ClockConfigPanel } from "@/components/chess/ClockConfigPanel";
import { ActionBar } from "@/components/chess/ActionBar";
import { AnalysisSummary } from "@/components/chess/AnalysisSummary";
import { PgnImportModal } from "@/components/chess/PgnImportModal";
import type { ClockConfig } from "@/hooks/useChessClock";
import { minutesToMs } from "@/hooks/useChessClock";
import { useChessKeyboard } from "@/hooks/useChessKeyboard";
import { useMultiPvControl } from "@/hooks/useMultiPvControl";
import type { AiAnalysisResult } from "@/wasm/generated/wasm-contract";
import type { ChessColor } from "@/types/chess";
import type { ChessDifficulty, ChessGameMode, ChessSearchMode } from "./Chess.types";
import { useChess } from "./Chess.hooks";
import styles from "./Chess.module.css";

const RESULT_TEXT = {
  "white-wins": "Brancas vencem!",
  "black-wins": "Pretas vencem!",
  draw: "Empate!",
};

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

  useChessKeyboard({
    currentPly,
    historyLength: history.length,
    jumpToPly,
    pendingPromotion,
  });

  useMultiPvControl({
    isAnalysisMode,
    analysisEnabled,
    maxLines,
    analysisTimeMs,
    result,
    isAtLatest,
    currentPly,
    runMultiPv,
    stopMultiPv,
  });

  return (
    <div className={styles.page}>
      <div className={styles.layout}>
        {/* ── Left: board area ─────────────────────────── */}
        <div className={styles.boardArea}>
          <ModeSelector mode={mode} onModeChange={handleModeChange} />

          {mode === "human-vs-ai" && (
            <AiSetupPanel
              mode="human-vs-ai"
              aiColor={aiColor}
              onAiColorChange={(c) => {
                setAiColor(c);
                setFlipped(c === "white");
              }}
              searchMode={searchMode}
              onSearchModeChange={setSearchMode}
              difficulty={difficulty}
              onDifficultyChange={setDifficulty}
              customTimeMs={customTimeMs}
              onCustomTimeMsChange={setCustomTimeMs}
              customDepth={customDepth}
              onCustomDepthChange={setCustomDepth}
              onRestart={restartGame}
              showColorChoice
            />
          )}

          {mode === "ai-vs-ai" && (
            <AiSetupPanel
              mode="ai-vs-ai"
              aiColor={aiColor}
              onAiColorChange={setAiColor}
              searchMode={searchMode}
              onSearchModeChange={setSearchMode}
              difficulty={difficulty}
              onDifficultyChange={setDifficulty}
              customTimeMs={customTimeMs}
              onCustomTimeMsChange={setCustomTimeMs}
              customDepth={customDepth}
              onCustomDepthChange={setCustomDepth}
              onRestart={restartGame}
              showColorChoice={false}
            />
          )}

          <TurnBanner
            currentPlayer={currentPlayer}
            result={result}
            resultText={resultText}
            aiThinking={aiThinking}
            checkSquare={checkSquare}
            isAtLatest={isAtLatest}
          />

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

          <ActionBar
            flipped={flipped}
            onFlip={() => setFlipped((f) => !f)}
            onRestart={handleRestart}
            copyStatus={copyStatus}
            onCopyPgn={handleCopyPgn}
            onPastePgn={() => {
              setPgnText("");
              setPgnError(null);
              setPgnModalOpen(true);
            }}
            canAnalyze={!result || !isAtLatest}
            analyzing={analyzing || aiThinking}
            onAnalyze={handleAnalyze}
            autoAnalyze={autoAnalyze}
            onToggleAutoAnalyze={() => setAutoAnalyze(!autoAnalyze)}
            showAnalysisToggle={!result || isAnalysisMode}
            analysisEnabled={analysisEnabled}
            onToggleAnalysis={() => setAnalysisEnabled((v) => !v)}
            hasHistory={history.length > 0}
          />

          {displayedAnalysis && (
            <AnalysisSummary
              analysis={displayedAnalysis}
              onClose={isAtLatest ? () => setAnalysis(null) : undefined}
            />
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
              analysisEnabled={analysisEnabled}
              onToggleAnalysis={() => setAnalysisEnabled((v) => !v)}
              maxLines={maxLines}
              onMaxLinesChange={setMaxLines}
              analysisTimeMs={analysisTimeMs}
              onAnalysisTimeChange={setAnalysisTimeMs}
            />
          ) : (
            <>
              <ClockConfigPanel
                clockMinutes={clockMinutes}
                onClockMinutesChange={setClockMinutes}
                clockIncrement={clockIncrement}
                onClockIncrementChange={setClockIncrement}
                clockConfig={clockConfig}
                clock={clock}
                gameStarted={gameStarted}
              />

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
        <PgnImportModal
          pgnText={pgnText}
          onPgnTextChange={(text) => {
            setPgnText(text);
            setPgnError(null);
          }}
          pgnError={pgnError}
          onLoad={handleLoadPgn}
          onClose={() => setPgnModalOpen(false)}
        />
      )}
    </div>
  );
};