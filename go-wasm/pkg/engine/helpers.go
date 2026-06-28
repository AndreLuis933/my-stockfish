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

// oppositeColor returns the enemy color of the given piece color.
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

// Legacy free functions (delegate to Game) — used by the WASM bridge --------

// KingCheck returns the square index of the side-to-move king if it is in
// check, or -1 otherwise. Exposed to the frontend as `isCheckJS`.
func KingCheck() int {
	color := Game.colorOfSide()
	if Game.IsInCheck(color) {
		return Game.FindKing(color)
	}
	return -1
}

// Perft counts leaf nodes at the given depth from the Game position.
// Delegates to the Position method; kept for external callers.
func Perft(depth int) int {
	return Game.Perft(depth)
}