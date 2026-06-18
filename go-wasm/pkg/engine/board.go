package engine

import "webassemble/pkg/types"

var Board types.Board

const boardSize = 8

var enPassantCapture = -1
var enPassantTarget = -1
var whiteToMove bool
var castlingRights types.CastlingRights

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

func oppositeColor(color types.Piece) types.Piece {
	if color&types.ColorMask == types.ColorBlack {
		return types.ColorWhite
	}
	return types.ColorBlack

}

func PieceColor(color types.Piece) types.Piece {
	if color&types.ColorMask == types.ColorBlack {
		return types.ColorBlack
	}
	return types.ColorWhite

}

func isWhite(white bool) types.Piece {
	if white {
		return types.ColorWhite
	} else {
		return types.ColorBlack
	}
}

func KingCheck() int {
	color := isWhite(whiteToMove)
	if IsInCheck(color) {
		return FindKing(color)
	}
	return -1
}
