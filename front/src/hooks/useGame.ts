import { useEffect, useState } from "react";
import type { Board, Color, Move } from "@/types/game";
import {
  applyMove,
  checkResult,
  computeTurnState,
  initBoard,
  validMoves,
} from "@/utils/gameEngine";
import type { GameResult, TurnState } from "@/utils/gameEngine";
import { pickBestMove } from "@/utils/aiEngine";

export type GameMode = "human-vs-human" | "human-vs-ai" | "ai-vs-ai";

const AI_DELAY_MS = 350;

interface GameState {
  board: Board;
  currentPlayer: Color;
  selectedSquare: [number, number] | null;
  movesForSelected: Move[];
  turnState: TurnState;
  flashSelectable: boolean;
  movesSinceCapture: number;
  result: GameResult;
}

const initialState = (): GameState => {
  const board = initBoard();
  const turnState = computeTurnState(board, "white");
  return {
    board,
    currentPlayer: "white",
    selectedSquare: null,
    movesForSelected: [],
    turnState,
    flashSelectable: false,
    movesSinceCapture: 0,
    result: null,
  };
};

const isAiColor = (mode: GameMode, color: Color): boolean => {
  if (mode === "ai-vs-ai") return true;
  if (mode === "human-vs-ai") return color === "black";
  return false;
};

export const useGame = (mode: GameMode = "human-vs-ai") => {
  const [state, setState] = useState<GameState>(initialState);

  // Destructure at hook level so the effect deps list individual stable references
  // rather than the `state` object itself.
  const { board, currentPlayer, movesSinceCapture, result } = state;

  useEffect(() => {
    if (result !== null) return;
    if (!isAiColor(mode, currentPlayer)) return;

    const timer = setTimeout(() => {
      const best = pickBestMove(board, currentPlayer, movesSinceCapture);
      if (!best) return;

      const { from, move } = best;
      const movedPiece = board[from[0]][from[1]];
      const nextBoard = applyMove(from, move, board);
      const nextPlayer: Color = currentPlayer === "white" ? "black" : "white";
      const nextTurnState = computeTurnState(nextBoard, nextPlayer);
      const resetsCounter = move.captured.length > 0 || movedPiece?.type === "man";
      const nextMovesSinceCapture = resetsCounter ? 0 : movesSinceCapture + 1;

      setState({
        board: nextBoard,
        currentPlayer: nextPlayer,
        selectedSquare: null,
        movesForSelected: [],
        turnState: nextTurnState,
        flashSelectable: false,
        movesSinceCapture: nextMovesSinceCapture,
        result: checkResult(nextTurnState, nextPlayer, nextMovesSinceCapture),
      });
    }, AI_DELAY_MS);

    return () => clearTimeout(timer);
  }, [result, currentPlayer, board, movesSinceCapture, mode]);

  const handleSquareClick = (row: number, col: number) => {
    const {
      board: b,
      currentPlayer: cp,
      selectedSquare,
      movesForSelected,
      turnState,
      movesSinceCapture: msc,
      result: r,
    } = state;

    if (r !== null) return;
    if (isAiColor(mode, cp)) return;

    const targetMove = movesForSelected.find((m) => m.to[0] === row && m.to[1] === col);
    if (targetMove && selectedSquare) {
      const movedPiece = b[selectedSquare[0]][selectedSquare[1]];
      const nextBoard = applyMove(selectedSquare, targetMove, b);
      const nextPlayer: Color = cp === "white" ? "black" : "white";
      const nextTurnState = computeTurnState(nextBoard, nextPlayer);
      const resetsCounter = targetMove.captured.length > 0 || movedPiece?.type === "man";
      const nextMovesSinceCapture = resetsCounter ? 0 : msc + 1;
      setState({
        board: nextBoard,
        currentPlayer: nextPlayer,
        selectedSquare: null,
        movesForSelected: [],
        turnState: nextTurnState,
        flashSelectable: false,
        movesSinceCapture: nextMovesSinceCapture,
        result: checkResult(nextTurnState, nextPlayer, nextMovesSinceCapture),
      });
      return;
    }

    const piece = b[row][col];
    if (piece?.color === cp) {
      const isSelectable = turnState.selectable.some(([sr, sc]) => sr === row && sc === col);
      if (!isSelectable) {
        setState({ ...state, flashSelectable: true, selectedSquare: null, movesForSelected: [] });
        return;
      }
      setState({
        ...state,
        selectedSquare: [row, col],
        movesForSelected: validMoves(row, col, b, turnState.globalMax),
        flashSelectable: false,
      });
      return;
    }

    setState({ ...state, selectedSquare: null, movesForSelected: [], flashSelectable: false });
  };

  const restartGame = () => setState(initialState());

  return { state, handleSquareClick, restartGame };
};
