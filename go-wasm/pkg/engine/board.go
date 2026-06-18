package engine

import "webassemble/pkg/types"

var Board types.Board

const boardSize = 8

var enPassantCapture = -1
var enPassantTarget = -1

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func inBounds(idx int) bool {
	return idx >= 0 && idx < boardSize*boardSize
}

func PiecePtr(p types.Piece) *types.Piece { return &p }