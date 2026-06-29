package engine

import "webassemble/pkg/types"

// MoveKing generates one-step king moves using precomputed bitboard attack
// tables, plus castling (kingside & queenside).
//
// Castling checks all 6 FIDE conditions:
//  1. Castling rights still present (tracked in p.CastlingRights)
//  2. King and rook on their original squares
//  3. Squares between king and rook are empty
//  4. King is not currently in check
//  5. King does not pass through an attacked square
//  6. King does not land on an attacked square
//
// The rook move itself is applied in MakeMove (not here).
func (p *Position) MoveKing(piece types.Piece, i int, ml *MoveList) {
	var ownPieces Bitboard
	if piece.Color() == types.ColorWhite {
		ownPieces = p.WhitePieces
	} else {
		ownPieces = p.BlackPieces
	}

	targets := kingAttacks[i] & ^ownPieces

	for targets != 0 {
		to := bitscan(targets)
		targets &= targets - 1

		move := types.Move{From: uint8(i), To: uint8(to), Flag: types.FlagNormal}
		if captured := p.Board[to]; captured != 0 {
			move.Captured = captured
		}
		ml.Add(move)
	}

	p.generateCastling(piece, ml)
}
