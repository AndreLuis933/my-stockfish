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

  // Restart when mode changes
  useEffect(() => {
    setState(initialState());
  }, [mode]);

  // Trigger AI move when it's the AI's turn
  useEffect(() => {
    if (state.result !== null) return;
    if (!isAiColor(mode, state.currentPlayer)) return;

    const { board, currentPlayer, movesSinceCapture } = state;

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
  }, [state.result, state.currentPlayer, state.board, state.movesSinceCapture, mode]);

  const handleSquareClick = (row: number, col: number) => {
    const { board, currentPlayer, selectedSquare, movesForSelected, turnState, movesSinceCapture, result } = state;

    if (result !== null) return;
    if (isAiColor(mode, currentPlayer)) return;

    // Clicking a valid destination: apply the move
    const targetMove = movesForSelected.find((m) => m.to[0] === row && m.to[1] === col);
    if (targetMove && selectedSquare) {
      const movedPiece = board[selectedSquare[0]][selectedSquare[1]];
      const nextBoard = applyMove(selectedSquare, targetMove, board);
      const nextPlayer: Color = currentPlayer === "white" ? "black" : "white";
      const nextTurnState = computeTurnState(nextBoard, nextPlayer);
      const resetsCounter = targetMove.captured.length > 0 || movedPiece?.type === "man";
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
      return;
    }

    // Clicking a friendly piece
    const piece = board[row][col];
    if (piece?.color === currentPlayer) {
      const isSelectable = turnState.selectable.some(([r, c]) => r === row && c === col);
      if (!isSelectable) {
        setState({ ...state, flashSelectable: true, selectedSquare: null, movesForSelected: [] });
        return;
      }
      setState({
        ...state,
        selectedSquare: [row, col],
        movesForSelected: validMoves(row, col, board, turnState.globalMax),
        flashSelectable: false,
      });
      return;
    }

    // Clicking anything else: deselect
    setState({ ...state, selectedSquare: null, movesForSelected: [], flashSelectable: false });
  };

  const restartGame = () => setState(initialState());

  return { state, handleSquareClick, restartGame };
};
