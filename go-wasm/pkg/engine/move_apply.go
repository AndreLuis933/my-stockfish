package engine

import "webassemble/pkg/types"

func MakeMove(from, to, promotion int) {
	var piece = Board[from]
	var pieceTo = Board[to]
	if piece&types.Pawn == types.Pawn && to == enPassantTarget && enPassantCapture != -1 {
		Board[enPassantCapture] = 0
	}
	Board[from] = 0
	if promotion != 0 {
		Board[to] = types.Piece(promotion)
	} else {
		Board[to] = piece
	}

	whiteToMove = !whiteToMove

	enPassantCapture, enPassantTarget = -1, -1

	if piece&types.Pawn == types.Pawn {
		diff := to - from
		if diff == 2*boardSize || diff == -2*boardSize {
			enPassantCapture = to
			enPassantTarget = (from + to) / 2
		}
	}

	if piece&types.King == types.King {

		if piece.Color() == types.ColorWhite {
			castlingRights &^= types.CastleWhiteAll
		} else {
			castlingRights &^= types.CastleBlackAll
		}
		dif := to - from
		switch dif {
		case 2:
			var rook = Board[to+1]
			Board[to+1] = 0
			Board[to-1] = rook
		case -2:
			var rook = Board[to-2]
			Board[to-2] = 0
			Board[to+1] = rook
		}
	}
	if piece&types.Rook == types.Rook || pieceTo&types.Rook == types.Rook {
		if from == 0 || to == 0 {
			castlingRights &^= types.CastleWhiteQ
		} else if from == 7 || to == 7 {
			castlingRights &^= types.CastleWhiteK
		} else if from == 56 || to == 56 {
			castlingRights &^= types.CastleBlackQ
		} else if from == 63 || to == 63 {
			castlingRights &^= types.CastleBlackK
		}
	}
}
