package engine

import "webassemble/pkg/types"

// MoveRook generates rook moves using magic bitboard lookup.
func (p *Position) MoveRook(piece types.Piece, i int, ml *MoveList) {
	var ownPieces Bitboard
	if piece.Color() == types.ColorWhite {
		ownPieces = p.WhitePieces
	} else {
		ownPieces = p.BlackPieces
	}

	targets := rookAttacksBB(i, p.Occupied) & ^ownPieces

	for targets != 0 {
		to := bitscan(targets)
		targets &= targets - 1

		move := types.Move{From: i, To: to, Flag: types.FlagNormal}
		if captured := p.Board[to]; captured != 0 {
			move.Captured = captured
		}
		ml.Add(move)
	}
}