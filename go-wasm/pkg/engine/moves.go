package engine

import "webassemble/pkg/types"

func GetValidMoves() []types.Move {
	var moves []types.Move
	for i, piece := range Board {

		if piece&types.Pawn == types.Pawn {
			moves = GetMovePawn(piece, i, moves)
		}
		if piece&types.Rook == types.Rook {
			moves = GetMoveRook(piece, i, moves)
		}
		if piece&types.Bishop == types.Bishop {
			moves = GetMoveBishop(piece, i, moves)
		}
		if piece&types.Queen == types.Queen {
			moves = GetMoveRook(piece, i, moves)
			moves = GetMoveBishop(piece, i, moves)
		}
		if piece&types.King == types.King {
			moves = GetMoveKing(piece, i, moves)
		}
		if piece&types.Knight == types.Knight {
			moves = GetMoveKnight(piece, i, moves)
		}

	}
	return moves
}