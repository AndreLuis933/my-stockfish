package ai

import (
	"webassemble/pkg/engine"
	"webassemble/pkg/types"
)

const (
	pawnValue   = 100
	knightValue = 320
	bishopValue = 330
	rookValue   = 500
	queenValue  = 900
	kingValue   = 20000
	winScore    = 100_000
)

func materialValue(piece types.Piece) int {
	switch piece & types.TypeMask {
	case types.Pawn:
		return pawnValue
	case types.Knight:
		return knightValue
	case types.Bishop:
		return bishopValue
	case types.Rook:
		return rookValue
	case types.Queen:
		return queenValue
	case types.King:
		return kingValue
	}
	return 0
}

// Piece-square tables: positional bonuses for each piece type on each of
// the 64 squares. White's perspective; for black, mirror vertically via
// index ^ 56. Values from the Chess Programming Wiki (simplified eval).
//
// Index 0 = a8 (top-left from white's view), index 63 = h1 (bottom-right).
// Positive = good square for that piece type.

var pawnTable = [64]int{
	0, 0, 0, 0, 0, 0, 0, 0,
	50, 50, 50, 50, 50, 50, 50, 50,
	10, 10, 20, 30, 30, 20, 10, 10,
	5, 5, 10, 25, 25, 10, 5, 5,
	0, 0, 0, 20, 20, 0, 0, 0,
	5, -5, -10, 0, 0, -10, -5, 5,
	5, 10, 10, -20, -20, 10, 10, 5,
	0, 0, 0, 0, 0, 0, 0, 0,
}

var knightTable = [64]int{
	-50, -40, -30, -30, -30, -30, -40, -50,
	-40, -20, 0, 0, 0, 0, -20, -40,
	-30, 0, 10, 15, 15, 10, 0, -30,
	-30, 5, 15, 20, 20, 15, 5, -30,
	-30, 0, 15, 20, 20, 15, 0, -30,
	-30, 5, 10, 15, 15, 10, 5, -30,
	-40, -20, 0, 5, 5, 0, -20, -40,
	-50, -40, -30, -30, -30, -30, -40, -50,
}

var bishopTable = [64]int{
	-20, -10, -10, -10, -10, -10, -10, -20,
	-10, 0, 0, 0, 0, 0, 0, -10,
	-10, 0, 5, 10, 10, 5, 0, -10,
	-10, 5, 5, 10, 10, 5, 5, -10,
	-10, 0, 10, 10, 10, 10, 0, -10,
	-10, 10, 10, 10, 10, 10, 10, -10,
	-10, 5, 0, 0, 0, 0, 5, -10,
	-20, -10, -10, -10, -10, -10, -10, -20,
}

var rookTable = [64]int{
	0, 0, 0, 0, 0, 0, 0, 0,
	5, 10, 10, 10, 10, 10, 10, 5,
	-5, 0, 0, 0, 0, 0, 0, -5,
	-5, 0, 0, 0, 0, 0, 0, -5,
	-5, 0, 0, 0, 0, 0, 0, -5,
	-5, 0, 0, 0, 0, 0, 0, -5,
	-5, 0, 0, 0, 0, 0, 0, -5,
	0, 0, 0, 5, 5, 0, 0, 0,
}

var queenTable = [64]int{
	-20, -10, -10, -5, -5, -10, -10, -20,
	-10, 0, 0, 0, 0, 0, 0, -10,
	-10, 0, 5, 5, 5, 5, 0, -10,
	-5, 0, 5, 5, 5, 5, 0, -5,
	0, 0, 5, 5, 5, 5, 0, -5,
	-10, 5, 5, 5, 5, 5, 0, -10,
	-10, 0, 5, 0, 0, 0, 0, -10,
	-20, -10, -10, -5, -5, -10, -10, -20,
}

var kingTable = [64]int{
	-30, -40, -40, -50, -50, -40, -40, -30,
	-30, -40, -40, -50, -50, -40, -40, -30,
	-30, -40, -40, -50, -50, -40, -40, -30,
	-30, -40, -40, -50, -50, -40, -40, -30,
	-20, -30, -30, -40, -40, -30, -30, -20,
	-10, -20, -20, -20, -20, -20, -20, -10,
	20, 20, 0, 0, 0, 0, 20, 20,
	20, 30, 10, 0, 0, 10, 30, 20,
}

func pstValue(piece types.Piece, square int) int {
	isWhite := piece&types.ColorMask == types.ColorWhite
	idx := square
	if !isWhite {
		idx = square ^ 56 // mirror vertically for black
	}
	switch piece & types.TypeMask {
	case types.Pawn:
		return pawnTable[idx]
	case types.Knight:
		return knightTable[idx]
	case types.Bishop:
		return bishopTable[idx]
	case types.Rook:
		return rookTable[idx]
	case types.Queen:
		return queenTable[idx]
	case types.King:
		return kingTable[idx]
	}
	return 0
}

// Evaluate returns a score from the perspective of the side to move.
// Positive = favorable for the side to move; negative = unfavorable.
//
// Combines material values with piece-square table positional bonuses.
// This is a static evaluation — it does not look ahead.
func Evaluate(p *engine.Position) int {
	score := 0
	for i, piece := range p.Board {
		if piece == 0 {
			continue
		}
		total := materialValue(piece) + pstValue(piece, i)

		pieceIsWhite := piece&types.ColorMask == types.ColorWhite
		if pieceIsWhite == p.WhiteToMove {
			score += total
		} else {
			score -= total
		}
	}
	return score
}