package engine

import "webassemble/pkg/types"

// MoveKnight generates knight moves using precomputed bitboard attack tables.
func (p *Position) MoveKnight(piece types.Piece, i int, ml *MoveList) {
	var ownPieces Bitboard
	if piece.Color() == types.ColorWhite {
		ownPieces = p.WhitePieces
	} else {
		ownPieces = p.BlackPieces
	}

	targets := knightAttacks[i] & ^ownPieces

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