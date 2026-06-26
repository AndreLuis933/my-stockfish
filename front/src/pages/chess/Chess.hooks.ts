import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { ChessBoard, ChessColor } from "@/types/chess";
import { emptyBoard } from "@/utils/chessEngine";
import {
  historyToPgn,
  parsePgn,
  stripCheckSuffix,
  toSan,
  type MoveData,
} from "@/utils/chessNotation";
import type { ClockConfig, UseChessClock } from "@/hooks/useChessClock";
import { useChessClock } from "@/hooks/useChessClock";
import type { WasmEngine, AiAnalysisResult, MultiPvLine } from "@/wasm/generated/wasm-contract";
import { useWasm } from "@/wasm/useWasm";
import { pvToSan, type PvSanEntry } from "@/utils/pvToSan";

export type ChessGameMode = "human-vs-ai" | "human-vs-human" | "ai-vs-ai" | "analysis";
export type ChessResult = "white-wins" | "black-wins" | "draw" | null;
export type ChessDifficulty = "easy" | "medium" | "hard";
export type ChessSearchMode = "difficulty" | "time" | "depth";

export interface PendingPromotion {
  from: number;
  to: number;
  options: number[];
}

export interface HistoryEntry {
  san: string;
  color: ChessColor;
  from: number;
  to: number;
  promotion?: number;
  boardBefore: ChessBoard;
  boardAfter: ChessBoard;
  checkSquareAfter: number | null;
  isCheckmate: boolean;
  analysis?: AiAnalysisResult | null;
}

interface Move {
  from: number;
  to: number;
  promotion?: number;
}

interface ChessState {
  board: ChessBoard;
  currentPlayer: ChessColor;
  selectedSquare: number | null;
  validMoveSquares: number[];
  result: ChessResult;
  pendingPromotion: PendingPromotion | null;
  candidateMoves: Move[];
  checkSquare: number | null;
  aiThinking: boolean;
  lastMove: { from: number; to: number } | null;
  boardBefore: ChessBoard | null;
  animateId: number;
}

const initialState = (): ChessState => ({
  board: emptyBoard(),
  currentPlayer: "white",
  selectedSquare: null,
  validMoveSquares: [],
  result: null,
  pendingPromotion: null,
  candidateMoves: [],
  checkSquare: null,
  aiThinking: false,
  lastMove: null,
  boardBefore: null,
  animateId: 0,
});

const pieceColor = (byte: number): ChessColor | null => {
  if (!byte) return null;
  return (byte & 0b11000000) === 0b01000000 ? "white" : "black";
};

const toResult = (status: string): ChessResult => {
  if (status === "white-wins" || status === "black-wins" || status === "draw") {
    return status;
  }
  return null;
};

const DIFFICULTY_MS: Record<ChessDifficulty, number> = {
  easy: 100,
  medium: 500,
  hard: 2000,
};

const AI_DELAY_MS = 400;

const FILES = "abcdefgh";

const PROMOTION_LETTER_TO_BYTE: Record<string, number> = {
  Q: 0b00010000,
  R: 0b00001000,
  B: 0b00000100,
  N: 0b00000010,
};

const matchSanToLegal = (
  san: string,
  legalMoves: { from: number; to: number; promotion?: number }[],
  color: ChessColor,
): { from: number; to: number; promotion?: number } | null => {
  const clean = stripCheckSuffix(san).trim();

  if (clean === "O-O" || clean === "0-0") {
    const fromRank = color === "white" ? 0 : 7;
    const kingFrom = fromRank * 8 + 4;
    const kingTo = fromRank * 8 + 6;
    return (
      legalMoves.find((m) => m.from === kingFrom && m.to === kingTo) ?? null
    );
  }
  if (clean === "O-O-O" || clean === "0-0-0") {
    const fromRank = color === "white" ? 0 : 7;
    const kingFrom = fromRank * 8 + 4;
    const kingTo = fromRank * 8 + 2;
    return (
      legalMoves.find((m) => m.from === kingFrom && m.to === kingTo) ?? null
    );
  }

  const promoMatch = clean.match(/=([QRBN])$/);
  const promoLetter = promoMatch?.[1];
  let working = promoMatch ? clean.slice(0, -promoMatch[0].length) : clean;

  const toSquare = working.slice(-2);
  const toIdx = FILES.indexOf(toSquare[0]);
  if (toIdx === -1) return null;
  const toRank = parseInt(toSquare[1], 10);
  if (isNaN(toRank) || toRank < 1 || toRank > 8) return null;
  const to = (8 - toRank) * 8 + toIdx;

  working = working.slice(0, -2);

  const isCapture = working.endsWith("x");
  if (isCapture) working = working.slice(0, -1);

  const pieceMatch = working.match(/^[NBRQK]/);
  const pieceLetter = pieceMatch?.[0];
  let disamb = working;
  if (pieceLetter) disamb = working.slice(1);

  const candidates = legalMoves.filter((m) => m.to === to);

  const targetPromo = promoLetter
    ? (PROMOTION_LETTER_TO_BYTE[promoLetter] ?? 0) |
      (color === "white" ? 0b01000000 : 0b10000000)
    : undefined;

  const promoFiltered = promoLetter
    ? candidates.filter((m) => (m.promotion ?? 0) === (targetPromo ?? 0))
    : candidates.filter((m) => !m.promotion);

  if (promoFiltered.length === 0) return null;

  if (!pieceLetter || pieceLetter === "P") {
    if (!disamb) return promoFiltered[0] ?? null;
    if (disamb.length === 1) {
      const c = disamb;
      if (FILES.includes(c)) {
        const fromFile = FILES.indexOf(c);
        return (
          promoFiltered.find((m) => Math.floor(m.from / 8) !== -1 && m.from % 8 === fromFile) ??
          null
        );
      }
      const rankNum = parseInt(c, 10);
      if (!isNaN(rankNum)) {
        const fromRankIdx = 8 - rankNum;
        return (
          promoFiltered.find((m) => Math.floor(m.from / 8) === fromRankIdx) ??
          null
        );
      }
    }
  }

  return promoFiltered[0] ?? null;
};

export interface UseChessReturn {
  state: ChessState;
  history: HistoryEntry[];
  currentPly: number;
  isAtLatest: boolean;
  handleSquareClick: (index: number) => Promise<void>;
  restartGame: () => Promise<void>;
  choosePromotion: (promotionByte: number) => Promise<void>;
  cancelPromotion: () => void;
  jumpToPly: (ply: number) => void;
  clock: UseChessClock;
  gameStarted: boolean;
  analyze: (timeLimitMs: number) => Promise<AiAnalysisResult | null>;
  autoAnalyze: boolean;
  setAutoAnalyze: (on: boolean) => void;
  analyzeCurrentPosition: () => Promise<void>;
  analysisForPly: (ply: number) => AiAnalysisResult | null | undefined;
  exportPgn: () => string;
  loadPgn: (pgn: string) => Promise<boolean>;
  multiPv: MultiPvLine[];
  multiPvSan: PvSanEntry[][];
  multiPvDepth: number;
  multiPvThinking: boolean;
  currentFen: string;
  runMultiPv: (numLines: number, timeMs: number) => void;
  stopMultiPv: () => void;
}

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

  const isAiTurn = useCallback(
    (color: ChessColor): boolean => {
      if (mode === "ai-vs-ai") return true;
      if (mode === "human-vs-ai") return color === aiColor;
      return false;
    },
    [mode, aiColor],
  );

  const isAtLatest = currentPly === history.length;

  // ── Clock ──────────────────────────────────────────────
  const clock = useChessClock(
    clockConfig,
    currentPlayer,
    gameStarted && isAtLatest,
    result !== null,
  );

  // Flag fall → game over
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

    return () => {
      cancelled = true;
    };
  }, [engine]);

  // ── Helper: apply a move and record history ────────────
  const onMoveCompleteRef = useRef(clock.onMoveComplete);
  useEffect(() => {
    onMoveCompleteRef.current = clock.onMoveComplete;
  }, [clock.onMoveComplete]);

  const applyMove = useCallback(
    async (
      currentEngine: WasmEngine,
      from: number,
      to: number,
      promotion: number,
      moverColor: ChessColor,
    ): Promise<void> => {
      const boardBefore = stateRef.current.board.slice();
      const rawBoard = await currentEngine.makeMove(from, to, promotion);
      const boardAfter = Array.from(rawBoard) as ChessBoard;
      const checkSquare = await currentEngine.isCheckJS();
      const status = await currentEngine.gameStatus();
      const checkSq = checkSquare === -1 ? null : checkSquare;
      const gameResult = toResult(status);
      const isCheckmate = gameResult !== null && gameResult !== "draw";

      const moveData: MoveData = { from, to, promotion: promotion || undefined };
      const san = toSan(boardBefore, moveData, moverColor, checkSq, isCheckmate);

      const nextPlayer: ChessColor = moverColor === "white" ? "black" : "white";

      const entry: HistoryEntry = {
        san,
        color: moverColor,
        from,
        to,
        promotion: promotion || undefined,
        boardBefore,
        boardAfter,
        checkSquareAfter: checkSq,
        isCheckmate,
      };

      setHistory((h) => [...h, entry]);
      setCurrentPly((p) => p + 1);
      setGameStarted(true);

      setState((p) => ({
        ...p,
        board: boardAfter,
        selectedSquare: null,
        validMoveSquares: [],
        candidateMoves: [],
        currentPlayer: nextPlayer,
        checkSquare: checkSq,
        result: gameResult,
        lastMove: { from, to },
        boardBefore,
        animateId: p.animateId + 1,
      }));

      onMoveCompleteRef.current(moverColor);
    },
    [],
  );

  // ── AI turn effect ─────────────────────────────────────
  useEffect(() => {
    if (!engine || result !== null) return;
    if (mode === "human-vs-human" || mode === "analysis") return;
    if (mode === "human-vs-ai" && currentPlayer !== aiColor) return;
    if (!isAtLatest) return;

    const currentEngine = engine;
    const moverColor = currentPlayer;
    let cancelled = false;

    const timer = setTimeout(async () => {
      setState((p) => ({ ...p, aiThinking: true }));

      let moveJson: string;
      if (searchMode === "depth") {
        moveJson = await currentEngine.aiMoveDepth(customDepth);
      } else if (searchMode === "time") {
        moveJson = await currentEngine.aiMove(customTimeMs);
      } else {
        moveJson = await currentEngine.aiMove(DIFFICULTY_MS[difficulty]);
      }
      const aiMove: Move = JSON.parse(moveJson);

      if (cancelled) return;

      await applyMove(
        currentEngine,
        aiMove.from,
        aiMove.to,
        aiMove.promotion ?? 0,
        moverColor,
      );

      if (!cancelled) {
        setState((p) => ({ ...p, aiThinking: false }));
      }
    }, AI_DELAY_MS);

    return () => {
      cancelled = true;
      clearTimeout(timer);
      setState((p) => ({ ...p, aiThinking: false }));
    };
  }, [
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
  ]);

  // ── Square click handler ───────────────────────────────
  const handleSquareClick = useCallback(
    async (index: number) => {
      if (!engine) return;

      const prev = stateRef.current;

      if (prev.result !== null || prev.aiThinking || isAiTurn(prev.currentPlayer))
        return;
      if (!isAtLatest) return;

      const clickedColor = pieceColor(prev.board[index]);

      // Execute move if a valid target square is clicked
      if (
        prev.selectedSquare !== null &&
        prev.validMoveSquares.includes(index)
      ) {
        const from = prev.selectedSquare;
        const to = index;

        const promotionMoves = prev.candidateMoves.filter(
          (m) => m.from === from && m.to === to && m.promotion !== undefined,
        );

        if (promotionMoves.length > 0) {
          setState((p) => ({
            ...p,
            pendingPromotion: {
              from,
              to,
              options: promotionMoves.map((m) => m.promotion!),
            },
            selectedSquare: null,
            validMoveSquares: [],
          }));
          return;
        }

        await applyMove(engine, from, to, 0, prev.currentPlayer);
        return;
      }

      // Clicking empty square, opponent piece, or same piece → clear selection
      if (
        !clickedColor ||
        clickedColor !== prev.currentPlayer ||
        prev.selectedSquare === index
      ) {
        setState((p) => ({
          ...p,
          selectedSquare: null,
          validMoveSquares: [],
        }));
        return;
      }

      // Clicking a different own piece → select it and load its moves
      setState((p) => ({ ...p, selectedSquare: index, validMoveSquares: [] }));

      const movesJson = await engine.validMovesChess();
      const moves: Move[] = JSON.parse(movesJson);
      const ownMoves = moves.filter((m) => m.from === index);
      const targets = ownMoves.map((m) => m.to);

      setState((p) => {
        if (p.selectedSquare !== index) return p;
        return { ...p, validMoveSquares: targets, candidateMoves: ownMoves };
      });
    },
    [engine, isAiTurn, isAtLatest, applyMove],
  );

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

  // ── Promotion ──────────────────────────────────────────
  const choosePromotion = useCallback(
    async (promotionByte: number) => {
      const prev = stateRef.current;
      const pending = prev.pendingPromotion;
      if (!engine || !pending) return;

      await applyMove(
        engine,
        pending.from,
        pending.to,
        promotionByte,
        prev.currentPlayer,
      );

      setState((p) => ({ ...p, pendingPromotion: null }));
    },
    [engine, applyMove],
  );

  const cancelPromotion = useCallback(() => {
    setState((p) => ({ ...p, pendingPromotion: null }));
  }, []);

  // ── Analysis (evaluation + best move) ──────────────────
  const analyze = useCallback(
    async (timeLimitMs: number): Promise<AiAnalysisResult | null> => {
      if (!engine) return null;
      const json = await engine.aiAnalysis(timeLimitMs);
      return JSON.parse(json) as AiAnalysisResult;
    },
    [engine],
  );

  // ── Auto-analyze: run analysis after each move ──────────
  const [autoAnalyze, setAutoAnalyze] = useState(false);

  useEffect(() => {
    if (!engine || !autoAnalyze) return;
    if (history.length === 0) return;
    if (!isAtLatest) return;

    const latestPly = history.length;
    const latestEntry = history[latestPly - 1];
    if (latestEntry.analysis) return;
    if (latestEntry.isCheckmate) return;

    let cancelled = false;

    const timer = setTimeout(async () => {
      const result = await analyze(500);
      if (cancelled) return;
      setHistory((h) => {
        if (h.length < latestPly) return h;
        const updated = h.slice();
        updated[latestPly - 1] = { ...updated[latestPly - 1], analysis: result };
        return updated;
      });
    }, 200);

    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [engine, autoAnalyze, history, isAtLatest, analyze]);

  // ── Analysis for a specific ply (from saved history) ──
  const analysisForPly = useCallback(
    (ply: number): AiAnalysisResult | null | undefined => {
      if (ply <= 0 || ply > history.length) return undefined;
      return history[ply - 1].analysis;
    },
    [history],
  );

  // ── Analyze current viewed position (saves into history) ──
  const analyzeCurrentPosition = useCallback(async () => {
    if (!engine) return;
    if (currentPly <= 0 || currentPly > historyRef.current.length) return;

    const ply = currentPly;
    const result = await analyze(1000);
    if (!result) return;

    setHistory((h) => {
      if (ply > h.length) return h;
      const updated = h.slice();
      updated[ply - 1] = { ...updated[ply - 1], analysis: result };
      return updated;
    });
  }, [engine, analyze, currentPly]);

  // ── Multi-PV continuous analysis (lichess-style live deepening) ──
  const [multiPv, setMultiPv] = useState<MultiPvLine[]>([]);
  const [multiPvThinking, setMultiPvThinking] = useState(false);
  const multiPvRef = useRef<{ cancelled: boolean } | null>(null);

  const multiPvSan = useMemo<PvSanEntry[][]>(() => {
    if (multiPv.length === 0) return [];
    const board = state.board.slice();
    const startColor: ChessColor = state.currentPlayer;
    return multiPv.map((line) => pvToSan(board, line.moves, startColor));
  }, [multiPv, state.board, state.currentPlayer]);

  const multiPvDepth = multiPv.length > 0 ? Math.max(...multiPv.map((l) => l.depth)) : 0;

  // Continuous Multi-PV loop: runs while in analysis mode, re-searches the
  // current position repeatedly so depth/score update live. Stops on
  // navigation, mode change, or unmount.
  const runMultiPv = useCallback(
    (numLines: number, timeMs: number) => {
      if (!engine) return;
      if (multiPvRef.current) multiPvRef.current.cancelled = true;
      const token = { cancelled: false };
      multiPvRef.current = token;
      const currentEngine = engine;

      setMultiPvThinking(true);
      setMultiPv([]);

      async function loop() {
        while (!token.cancelled) {
          const json = await currentEngine.aiMultiPv(timeMs, numLines);
          if (token.cancelled) break;
          const lines = JSON.parse(json) as MultiPvLine[];
          setMultiPv(lines);
        }
        if (!token.cancelled) {
          setMultiPvThinking(false);
        }
      }
      loop();
    },
    [engine],
  );

  const stopMultiPv = useCallback(() => {
    if (multiPvRef.current) {
      multiPvRef.current.cancelled = true;
      multiPvRef.current = null;
    }
    setMultiPvThinking(false);
  }, []);

  // ── Current FEN (live, from the engine) ──
  const [currentFen, setCurrentFen] = useState("");

  useEffect(() => {
    if (!engine) return;
    let cancelled = false;
    const currentEngine = engine;
    async function fetchFen() {
      const fen = await currentEngine.fen();
      if (!cancelled) setCurrentFen(fen);
    }
    fetchFen();
    return () => { cancelled = true; };
  }, [engine, state.board, currentPly]);

  // ── Export PGN of the current game ─────────────────────
  const exportPgn = useCallback((): string => {
    return historyToPgn(history, state.result);
  }, [history, state.result]);

  // ── Load a game from PGN (view-only) ───────────────────
  const loadPgn = useCallback(
    async (pgn: string): Promise<boolean> => {
      if (!engine) return false;
      const sanMoves = parsePgn(pgn);
      if (sanMoves.length === 0) return false;

      const currentEngine = engine;

      await currentEngine.initBoard();

      const rebuiltHistory: HistoryEntry[] = [];
      let moverColor: ChessColor = "white";
      let lastBoard = emptyBoard();
      let lastCheckSquare: number | null = null;
      let gameResult: ChessResult = null;

      for (const sanToken of sanMoves) {
        const movesJson = await currentEngine.validMovesChess();
        const legalMoves: Move[] = JSON.parse(movesJson);

        const matched = matchSanToLegal(
          sanToken,
          legalMoves,
          moverColor,
        );

        if (!matched) return false;

        const boardBefore = lastBoard;
        const rawBoard = await currentEngine.makeMove(
          matched.from,
          matched.to,
          matched.promotion ?? 0,
        );
        const boardAfter = Array.from(rawBoard) as ChessBoard;
        const checkSquare = await currentEngine.isCheckJS();
        const status = await currentEngine.gameStatus();
        const checkSq = checkSquare === -1 ? null : checkSquare;
        const result = toResult(status);
        const isCheckmate = result !== null && result !== "draw";

        rebuiltHistory.push({
          san: sanToken,
          color: moverColor,
          from: matched.from,
          to: matched.to,
          promotion: matched.promotion || undefined,
          boardBefore,
          boardAfter,
          checkSquareAfter: checkSq,
          isCheckmate,
        });

        lastBoard = boardAfter;
        lastCheckSquare = checkSq;
        gameResult = result;
        moverColor = moverColor === "white" ? "black" : "white";
      }

      setHistory(rebuiltHistory);
      setCurrentPly(rebuiltHistory.length);
      setGameStarted(true);
      setState((p) => ({
        ...p,
        board: lastBoard,
        selectedSquare: null,
        validMoveSquares: [],
        candidateMoves: [],
        currentPlayer: moverColor,
        checkSquare: lastCheckSquare,
        result: gameResult,
        lastMove:
          rebuiltHistory.length > 0
            ? {
                from: rebuiltHistory[rebuiltHistory.length - 1].from,
                to: rebuiltHistory[rebuiltHistory.length - 1].to,
              }
            : null,
        boardBefore:
          rebuiltHistory.length > 0
            ? rebuiltHistory[rebuiltHistory.length - 1].boardBefore
            : null,
        animateId: p.animateId + 1,
        pendingPromotion: null,
        aiThinking: false,
      }));

      return true;
    },
    [engine],
  );

  // ── Navigation (jump to ply) ───────────────────────────
  const jumpToPly = useCallback(
    (ply: number) => {
      const clamped = Math.max(0, Math.min(history.length, ply));
      setCurrentPly(clamped);

      if (clamped === 0) {
        const startBoard =
          history.length > 0 ? history[0].boardBefore : stateRef.current.board;
        setState((p) => ({
          ...p,
          board: startBoard,
          selectedSquare: null,
          validMoveSquares: [],
          candidateMoves: [],
          currentPlayer: "white" as ChessColor,
          checkSquare: null,
          lastMove: null,
          boardBefore: null,
        }));
      } else {
        const entry = history[clamped - 1];
        const nextPlayer: ChessColor = entry.color === "white" ? "black" : "white";
        setState((p) => ({
          ...p,
          board: entry.boardAfter,
          selectedSquare: null,
          validMoveSquares: [],
          candidateMoves: [],
          currentPlayer: nextPlayer,
          checkSquare: entry.checkSquareAfter,
          lastMove: { from: entry.from, to: entry.to },
          boardBefore: entry.boardBefore,
        }));
      }
    },
    [history],
  );

  const memoizedReturn = useMemo(
    () => ({
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
    }),
    [
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

  return memoizedReturn;
};