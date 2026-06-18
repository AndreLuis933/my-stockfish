package engine

import "webassemble/pkg/types"

func MakeMovement(from, to, promotion int) {
	var piece = Board[from]
	if piece&types.Pawn == types.Pawn && to == enPassantTarget && enPassantCapture != -1 {
		Board[enPassantCapture] = 0
	}
	Board[from] = 0
	if promotion != 0 {
		Board[to] = types.Piece(promotion)
	} else {
		Board[to] = piece

	}

	enPassantCapture, enPassantTarget = -1, -1

	if piece&types.Pawn == types.Pawn {
		diff := to - from
		if diff == 2*boardSize || diff == -2*boardSize {
			enPassantCapture = to
			enPassantTarget = (from + to) / 2
		}
	}
}
