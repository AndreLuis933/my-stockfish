import { useCallback, useEffect, useRef, useState } from "react";
import type { ChessBoard, ChessColor } from "@/types/chess";
import { emptyBoard } from "@/utils/chessEngine";
import { useWasm } from "@/wasm/useWasm";

export type ChessGameMode = "human-vs-ai" | "human-vs-human";
export type ChessResult = "white-wins" | "black-wins" | "draw" | null;
export type ChessDifficulty = "easy" | "medium" | "hard";
export type ChessSearchMode = "difficulty" | "time" | "depth";

export interface PendingPromotion {
  from: number;
  to: number;
  options: number[];
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

export const useChess = (
  mode: ChessGameMode = "human-vs-human",
  aiColor: ChessColor = "black",
  difficulty: ChessDifficulty = "medium",
  searchMode: ChessSearchMode = "difficulty",
  customTimeMs: number = 1000,
  customDepth: number = 4,
) => {
  const [state, setState] = useState<ChessState>(initialState);
  const stateRef = useRef(state);

  useEffect(() => {
    stateRef.current = state;
  });

  const { engine } = useWasm();

  const { currentPlayer, result } = state;

  const isAiTurn = useCallback(
    (color: ChessColor): boolean => mode === "human-vs-ai" && color === aiColor,
    [mode, aiColor],
  );

  // Load initial board
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
          board: Array.from(rawBoard),
          checkSquare: checkSquare === -1 ? null : checkSquare,
        }));
      }
    }

    loadBoard();

    return () => {
      cancelled = true;
    };
  }, [engine]);

  // AI turn effect — fires when it's the AI's turn and the engine is ready
  useEffect(() => {
    if (!engine || result !== null) return;
    if (mode !== "human-vs-ai") return;
    if (currentPlayer !== aiColor) return;

    const currentEngine = engine;
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

      const rawBoard = await currentEngine.makeMove(
        aiMove.from,
        aiMove.to,
        aiMove.promotion,
      );
      const checkSquare = await currentEngine.isCheckJS();
      const status = await currentEngine.gameStatus();
      const nextPlayer: ChessColor = aiColor === "white" ? "black" : "white";

      if (cancelled) return;

      setState((p) => ({
        ...p,
        board: Array.from(rawBoard),
        selectedSquare: null,
        validMoveSquares: [],
        candidateMoves: [],
        currentPlayer: nextPlayer,
        checkSquare: checkSquare === -1 ? null : checkSquare,
        result: toResult(status),
        aiThinking: false,
      }));
    }, AI_DELAY_MS);

    return () => {
      cancelled = true;
      clearTimeout(timer);
      setState((p) => ({ ...p, aiThinking: false }));
    };
  }, [engine, mode, aiColor, difficulty, searchMode, customTimeMs, customDepth, currentPlayer, result]);

  const handleSquareClick = useCallback(
    async (index: number) => {
      if (!engine) {
        console.log("sem engine");
        return;
      }

      const prev = stateRef.current;

      if (prev.result !== null || prev.aiThinking || isAiTurn(prev.currentPlayer)) return;

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

        const rawBoard = await engine.makeMove(from, to);
        const checkSquare = await engine.isCheckJS();
        const status = await engine.gameStatus();
        const nextPlayer: ChessColor = prev.currentPlayer === "white" ? "black" : "white";

        setState((p) => ({
          ...p,
          board: Array.from(rawBoard),
          selectedSquare: null,
          validMoveSquares: [],
          candidateMoves: [],
          currentPlayer: nextPlayer,
          checkSquare: checkSquare === -1 ? null : checkSquare,
          result: toResult(status),
        }));
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
    [engine, isAiTurn],
  );

  const restartGame = useCallback(async () => {
    setState(initialState);
    if (engine) {
      const rawBoard = await engine.initBoard();
      const checkSquare = await engine.isCheckJS();
      setState((prev) => ({
        ...prev,
        board: Array.from(rawBoard),
        checkSquare: checkSquare === -1 ? null : checkSquare,
      }));
    }
  }, [engine]);

  const choosePromotion = useCallback(
    async (promotionByte: number) => {
      const prev = stateRef.current;
      const pending = prev.pendingPromotion;
      if (!engine || !pending) return;

      const rawBoard = await engine.makeMove(
        pending.from,
        pending.to,
        promotionByte,
      );
      const checkSquare = await engine.isCheckJS();
      const status = await engine.gameStatus();
      const nextPlayer: ChessColor = prev.currentPlayer === "white" ? "black" : "white";

      setState((p) => ({
        ...p,
        board: Array.from(rawBoard),
        pendingPromotion: null,
        candidateMoves: [],
        currentPlayer: nextPlayer,
        checkSquare: checkSquare === -1 ? null : checkSquare,
        result: toResult(status),
      }));
    },
    [engine],
  );

  const cancelPromotion = useCallback(() => {
    setState((p) => ({ ...p, pendingPromotion: null }));
  }, []);

  return {
    state,
    handleSquareClick,
    restartGame,
    choosePromotion,
    cancelPromotion,
  };
};