import { useCallback, useEffect, useRef } from "react";
import type { ChessBoard, ChessColor, ChessMove, HistoryEntry } from "@/types/chess";
import type { WasmEngine } from "@/wasm/generated/wasm-contract";
import type { ChessState } from "@/pages/chess/Chess.types";
import { pieceColor, toResult } from "@/pages/chess/Chess.types";

interface UseChessMovesParams {
  engine: WasmEngine | null;
  stateRef: React.RefObject<ChessState>;
  setState: React.Dispatch<React.SetStateAction<ChessState>>;
  isAiTurn: (color: ChessColor) => boolean;
  isAtLatest: boolean;
  onMoveComplete: (color: ChessColor) => void;
  setHistory: React.Dispatch<React.SetStateAction<HistoryEntry[]>>;
  setCurrentPly: React.Dispatch<React.SetStateAction<number>>;
  setGameStarted: React.Dispatch<React.SetStateAction<boolean>>;
}

export interface UseChessMovesReturn {
  applyMove: (
    engine: WasmEngine,
    from: number,
    to: number,
    promotion: number,
    moverColor: ChessColor,
  ) => Promise<void>;
  handleSquareClick: (index: number) => Promise<void>;
  choosePromotion: (promotionByte: number) => Promise<void>;
  cancelPromotion: () => void;
}

export const useChessMoves = ({
  engine,
  stateRef,
  setState,
  isAiTurn,
  isAtLatest,
  onMoveComplete,
  setHistory,
  setCurrentPly,
  setGameStarted,
}: UseChessMovesParams): UseChessMovesReturn => {
  const onMoveCompleteRef = useRef(onMoveComplete);
  useEffect(() => {
    onMoveCompleteRef.current = onMoveComplete;
  }, [onMoveComplete]);

  const applyMove = useCallback(
    async (
      currentEngine: WasmEngine,
      from: number,
      to: number,
      promotion: number,
      moverColor: ChessColor,
    ): Promise<void> => {
      const boardBefore = stateRef.current.board.slice();
      const san = await currentEngine.san(from, to, promotion || undefined);
      const rawBoard = await currentEngine.makeMove(from, to, promotion);
      const boardAfter = Array.from(rawBoard) as ChessBoard;
      const checkSquare = await currentEngine.isCheckJS();
      const status = await currentEngine.gameStatus();
      const checkSq = checkSquare === -1 ? null : checkSquare;
      const gameResult = toResult(status);
      const isCheckmate = gameResult !== null && gameResult !== "draw";

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
    [stateRef, setState, setHistory, setCurrentPly, setGameStarted],
  );

  const handleSquareClick = useCallback(
    async (index: number) => {
      if (!engine) return;

      const prev = stateRef.current;

      if (prev.result !== null || prev.aiThinking || isAiTurn(prev.currentPlayer))
        return;
      if (!isAtLatest) return;

      const clickedColor = pieceColor(prev.board[index]);

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

      setState((p) => ({ ...p, selectedSquare: index, validMoveSquares: [] }));

      const movesJson = await engine.validMovesChess();
      const moves: ChessMove[] = JSON.parse(movesJson);
      const ownMoves = moves.filter((m) => m.from === index);
      const targets = ownMoves.map((m) => m.to);

      setState((p) => {
        if (p.selectedSquare !== index) return p;
        return { ...p, validMoveSquares: targets, candidateMoves: ownMoves };
      });
    },
    [engine, isAiTurn, isAtLatest, applyMove, stateRef, setState],
  );

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
    [engine, applyMove, stateRef, setState],
  );

  const cancelPromotion = useCallback(() => {
    setState((p) => ({ ...p, pendingPromotion: null }));
  }, [setState]);

  return { applyMove, handleSquareClick, choosePromotion, cancelPromotion };
};