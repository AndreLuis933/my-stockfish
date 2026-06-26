import { useCallback, useEffect, useState } from "react";
import type { ChessBoard, ChessColor, HistoryEntry } from "@/types/chess";
import { emptyBoard } from "@/types/chess";
import { historyToPgn } from "@/utils/chessNotation";
import type { WasmEngine, PgnHistoryEntry } from "@/wasm/generated/wasm-contract";
import type { ChessResult, ChessState } from "@/pages/chess/Chess.types";

interface UseChessHistoryParams {
  engine: WasmEngine | null;
  stateRef: React.RefObject<ChessState>;
  state: ChessState;
  setState: React.Dispatch<React.SetStateAction<ChessState>>;
  history: HistoryEntry[];
  setHistory: React.Dispatch<React.SetStateAction<HistoryEntry[]>>;
  currentPly: number;
  setCurrentPly: React.Dispatch<React.SetStateAction<number>>;
  setGameStarted: React.Dispatch<React.SetStateAction<boolean>>;
}

export interface UseChessHistoryReturn {
  jumpToPly: (ply: number) => void;
  exportPgn: () => string;
  loadPgn: (pgn: string) => Promise<boolean>;
  currentFen: string;
}

export const useChessHistory = ({
  engine,
  stateRef,
  state,
  setState,
  history,
  setHistory,
  currentPly,
  setCurrentPly,
  setGameStarted,
}: UseChessHistoryParams): UseChessHistoryReturn => {
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

  const exportPgn = useCallback((): string => {
    return historyToPgn(history, state.result);
  }, [history, state.result]);

  const loadPgn = useCallback(
    async (pgn: string): Promise<boolean> => {
      if (!engine) return false;

      const json = await engine.applyPgn(pgn);
      const entries = JSON.parse(json) as PgnHistoryEntry[];
      if (entries.length === 0) return false;

      const rebuiltHistory: HistoryEntry[] = [];
      let moverColor: ChessColor = "white";
      let lastBoard: ChessBoard = emptyBoard();
      let lastCheckSquare: number | null = null;
      let gameResult: ChessResult = null;

      for (const entry of entries) {
        const checkSq = entry.checkSquare === -1 ? null : entry.checkSquare;
        rebuiltHistory.push({
          san: entry.san,
          color: moverColor,
          from: entry.from,
          to: entry.to,
          promotion: entry.promotion || undefined,
          boardBefore: entry.boardBefore as ChessBoard,
          boardAfter: entry.boardAfter as ChessBoard,
          checkSquareAfter: checkSq,
          isCheckmate: entry.isCheckmate,
        });
        lastBoard = entry.boardAfter as ChessBoard;
        lastCheckSquare = checkSq;
        if (entry.isCheckmate) {
          gameResult = moverColor === "white" ? "white-wins" : "black-wins";
        }
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
    [engine, setHistory, setCurrentPly, setGameStarted, setState],
  );

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
    [history, setCurrentPly, setState, stateRef],
  );

  return { jumpToPly, exportPgn, loadPgn, currentFen };
};