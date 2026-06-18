package engine

import (
	"webassemble/pkg/types"
)

const boardSize = 8

// Pure helpers (no state) -------------------------------------------------

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func inBounds(idx int) bool {
	return idx >= 0 && idx < boardSize*boardSize
}

func oppositeColor(color types.Piece) types.Piece {
	if color&types.ColorMask == types.ColorBlack {
		return types.ColorWhite
	}
	return types.ColorBlack
}

// colorOfSide returns the Piece color bits for the side to move.
func (p *Position) colorOfSide() types.Piece {
	if p.WhiteToMove {
		return types.ColorWhite
	}
	return types.ColorBlack
}

// TODO: remove once move generators are migrated (duplicate of oppositeColor).
func pieceColor(color types.Piece) types.Piece {
	if color&types.ColorMask == types.ColorBlack {
		return types.ColorBlack
	}
	return types.ColorWhite
}

// Legacy helpers (to be deleted after full migration) --------------------

func PiecePtr(p types.Piece) *types.Piece { return &p }

// KingCheck returns the square index of the side-to-move king if it is in
// check, or -1 otherwise. Exposed to the frontend as `isCheckJS`.
func KingCheck() int {
	color := Game.colorOfSide()
	if Game.IsInCheck(color) {
		return Game.FindKing(color)
	}
	return -1
}

// Perft counts the number of leaf nodes at the given depth from the current
// Game position. Used for move-generation validation and as a performance
// baseline. Will move to a method on *Position and use Make/Unmake later.
func Perft(depth int) int {
	return Game.Perft(depth)
}