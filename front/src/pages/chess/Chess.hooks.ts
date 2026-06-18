import { useCallback, useEffect, useRef, useState } from "react";
import type { ChessBoard, ChessColor } from "@/types/chess";
import { emptyBoard } from "@/utils/chessEngine";
import { useWasm } from "@/wasm/useWasm";

export type ChessGameMode = "human-vs-ai" | "human-vs-human";
export type ChessResult = "white-wins" | "black-wins" | "draw" | null;

interface Move {
  from: number;
  to: number;
}

interface ChessState {
  board: ChessBoard;
  currentPlayer: ChessColor;
  selectedSquare: number | null;
  validMoveSquares: number[];
  result: ChessResult;
}

const initialState = (): ChessState => ({
  board: emptyBoard(),
  currentPlayer: "white",
  selectedSquare: null,
  validMoveSquares: [],
  result: null,
});

const pieceColor = (byte: number): ChessColor | null => {
  if (!byte) return null;
  return (byte & 0b11000000) === 0b01000000 ? "white" : "black";
};

const isAiTurn = (mode: ChessGameMode, color: ChessColor): boolean =>
  mode === "human-vs-ai" && color === "black";

export const useChess = (mode: ChessGameMode = "human-vs-ai") => {
  const [state, setState] = useState<ChessState>(initialState);
  const stateRef = useRef(state);

  useEffect(() => {
    stateRef.current = state;
  });

  const { engine } = useWasm();

  useEffect(() => {
    if (!engine) return;

    const currentEngine = engine;
    let cancelled = false;

    async function loadBoard() {
      const rawBoard = await currentEngine.initBoard();
      if (!cancelled) {
        setState((prev) => ({ ...prev, board: Array.from(rawBoard) }));
      }
    }

    loadBoard();

    return () => {
      cancelled = true;
    };
  }, [engine]);

  const handleSquareClick = useCallback(
    async (index: number) => {
      if (!engine) {
        console.log("sem engine");
        return;
      }

      const prev = stateRef.current;

      if (prev.result !== null || isAiTurn(mode, prev.currentPlayer)) return;

      const clickedColor = pieceColor(prev.board[index]);

      // Execute move if a valid target square is clicked
      if (
        prev.selectedSquare !== null &&
        prev.validMoveSquares.includes(index)
      ) {
        const from = prev.selectedSquare;
        const to = index;
        const rawBoard = await engine.makeMove(from, to);

        setState((p) => ({
          ...p,
          board: Array.from(rawBoard),
          selectedSquare: null,
          validMoveSquares: [],
          currentPlayer: p.currentPlayer === "white" ? "black" : "white",
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
      const targets = moves
        .filter((m) => m.from === index)
        .map((m) => m.to);

      setState((p) => {
        if (p.selectedSquare !== index) return p;
        return { ...p, validMoveSquares: targets };
      });
    },
    [engine, mode],
  );

  const restartGame = useCallback(async () => {
    setState(initialState);
    if (engine) {
      const rawBoard = await engine.initBoard();
      setState((prev) => ({ ...prev, board: Array.from(rawBoard) }));
    }
  }, [engine]);

  return { state, handleSquareClick, restartGame };
};
