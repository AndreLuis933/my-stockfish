import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { ChessBoard, ChessColor, HistoryEntry } from "@/types/chess";
import type { ClockConfig } from "@/hooks/useChessClock";
import { useChessClock } from "@/hooks/useChessClock";
import { useWasm } from "@/wasm/useWasm";
import { useMultiPv } from "@/hooks/useMultiPv";
import { useChessAnalysis } from "@/hooks/useChessAnalysis";
import { useChessMoves } from "@/hooks/useChessMoves";
import { useChessAi } from "@/hooks/useChessAi";
import { useChessHistory } from "@/hooks/useChessHistory";
import {
  type ChessState,
  type ChessGameMode,
  type ChessDifficulty,
  type ChessSearchMode,
  type ChessResult,
  type UseChessReturn,
  initialState,
} from "./Chess.types";


export const useChess = (
  mode: ChessGameMode = "human-vs-human",
  aiColor: ChessColor = "black",
  difficulty: ChessDifficulty = "medium",
  searchMode: ChessSearchMode = "difficulty",
  customTimeMs: number = 1000,
  customDepth: number = 4,
  clockConfig: ClockConfig,
): UseChessReturn => {
  const [state, setState] = useState<ChessState>(initialState);
  const [history, setHistory] = useState<HistoryEntry[]>([]);
  const [currentPly, setCurrentPly] = useState(0);
  const [gameStarted, setGameStarted] = useState(false);

  const stateRef = useRef(state);
  const historyRef = useRef(history);

  useEffect(() => {
    stateRef.current = state;
  });

  useEffect(() => {
    historyRef.current = history;
  });

  const { engine } = useWasm();
  const { currentPlayer, result } = state;
  const isAtLatest = currentPly === history.length;

  const isAiTurn = useCallback(
    (color: ChessColor): boolean => {
      if (mode === "ai-vs-ai") return true;
      if (mode === "human-vs-ai") return color === aiColor;
      return false;
    },
    [mode, aiColor],
  );

  // ── Clock ──────────────────────────────────────────────
  const clock = useChessClock(
    clockConfig,
    currentPlayer,
    gameStarted && isAtLatest,
    result !== null,
  );

  useEffect(() => {
    if (clock.flagFallen && stateRef.current.result === null) {
      const winner: ChessResult =
        clock.flagFallen === "white" ? "black-wins" : "white-wins";
      setState((p) => ({ ...p, result: winner }));
      clock.stop();
    }
  }, [clock.flagFallen, clock]);

  // ── Load initial board ─────────────────────────────────
  useEffect(() => {
    if (!engine) return;
    const currentEngine = engine;
    let cancelled = false;

    async function loadBoard() {
      const rawBoard = await currentEngine.initBoard();
      const checkSquare = await currentEngine.isCheckJS();
      if (!cancelled) {
        setState((prev) => ({
          ...prev,
          board: Array.from(rawBoard) as ChessBoard,
          checkSquare: checkSquare === -1 ? null : checkSquare,
        }));
      }
    }

    loadBoard();
    return () => { cancelled = true; };
  }, [engine]);

  // ── Moves (apply, click, promotion) ────────────────────
  const {
    applyMove,
    handleSquareClick,
    playMove,
    choosePromotion,
    cancelPromotion,
  } = useChessMoves({
    engine,
    stateRef,
    setState,
    isAiTurn,
    isAtLatest,
    onMoveComplete: clock.onMoveComplete,
    setHistory,
    setCurrentPly,
    setGameStarted,
  });

  // ── AI turn ────────────────────────────────────────────
  useChessAi({
    engine,
    mode,
    aiColor,
    difficulty,
    searchMode,
    customTimeMs,
    customDepth,
    currentPlayer,
    result,
    isAtLatest,
    applyMove,
    setState,
  });

  // ── History (navigation, PGN, FEN) ─────────────────────
  const { jumpToPly, exportPgn, loadPgn, currentFen } = useChessHistory({
    engine,
    stateRef,
    state,
    setState,
    history,
    setHistory,
    currentPly,
    setCurrentPly,
    setGameStarted,
  });

  // ── Restart ────────────────────────────────────────────
  const restartGame = useCallback(async () => {
    setState(initialState);
    setHistory([]);
    setCurrentPly(0);
    setGameStarted(false);
    clock.reset(clockConfig);
    if (engine) {
      const rawBoard = await engine.initBoard();
      const checkSquare = await engine.isCheckJS();
      setState((prev) => ({
        ...prev,
        board: Array.from(rawBoard) as ChessBoard,
        checkSquare: checkSquare === -1 ? null : checkSquare,
      }));
    }
  }, [engine, clock, clockConfig]);

  // ── Analysis ───────────────────────────────────────────
  const {
    analyze,
    autoAnalyze,
    setAutoAnalyze,
    analyzeCurrentPosition,
    analysisForPly,
  } = useChessAnalysis({
    engine,
    history,
    setHistory,
    currentPly,
    isAtLatest,
  });

  // ── Multi-PV ───────────────────────────────────────────
  const {
    multiPv,
    multiPvSan,
    multiPvDepth,
    multiPvThinking,
    runMultiPv,
    stopMultiPv,
  } = useMultiPv(engine, state.board, state.currentPlayer);

  return useMemo(
    () => ({
      state,
      history,
      currentPly,
      isAtLatest,
      handleSquareClick,
      playMove,
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
    }),
    [
      state,
      history,
      currentPly,
      isAtLatest,
      handleSquareClick,
      playMove,
      restartGame,
      choosePromotion,
      cancelPromotion,
      jumpToPly,
      clock,
      gameStarted,
      analyze,
      autoAnalyze,
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
    ],
  );
};