import { useState } from "react";
import type { Board, Color, Move } from "@/types/game";
import { applyMove, initBoard, validMoves } from "@/utils/gameEngine";

interface GameState {
  board: Board;
  currentPlayer: Color;
  selectedSquare: [number, number] | null;
  movesForSelected: Move[];
}

const initialState = (): GameState => ({
  board: initBoard(),
  currentPlayer: "white",
  selectedSquare: null,
  movesForSelected: [],
});

export const useGame = () => {
  const [state, setState] = useState<GameState>(initialState);

  const handleSquareClick = (row: number, col: number) => {
    const { board, currentPlayer, selectedSquare, movesForSelected } = state;

    // Clicking a valid destination: apply the move
    const targetMove = movesForSelected.find((m) => m.to[0] === row && m.to[1] === col);
    if (targetMove && selectedSquare) {
      setState({
        board: applyMove(selectedSquare, targetMove, board),
        currentPlayer: currentPlayer === "white" ? "black" : "white",
        selectedSquare: null,
        movesForSelected: [],
      });
      return;
    }

    // Clicking a friendly piece: select it
    const piece = board[row][col];
    if (piece?.color === currentPlayer) {
      setState({
        ...state,
        selectedSquare: [row, col],
        movesForSelected: validMoves(row, col, board),
      });
      return;
    }

    // Clicking anything else: deselect
    setState({ ...state, selectedSquare: null, movesForSelected: [] });
  };

  return { state, handleSquareClick };
};
