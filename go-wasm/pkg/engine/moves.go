package engine

import "webassemble/pkg/types"

// PseudoLegalMoves generates all moves that are legal *ignoring* whether
// they leave the own king in check. The filtering happens in LegalMoves.
//
// Uses bitscan loops over piece bitboards instead of scanning all 64 squares.
// Only iterates over actual pieces (~16 per side), not empty squares.
func (p *Position) PseudoLegalMoves(ml *MoveList) {
	ml.Clear()

	var pawns, knights, bishops, rooks, queens, king, ownPieces Bitboard
	var color types.Piece
	if p.WhiteToMove {
		pawns, knights, bishops, rooks, queens, king = p.WhitePawns, p.WhiteKnights, p.WhiteBishops, p.WhiteRooks, p.WhiteQueens, p.WhiteKing
		ownPieces = p.WhitePieces
		color = types.ColorWhite
	} else {
		pawns, knights, bishops, rooks, queens, king = p.BlackPawns, p.BlackKnights, p.BlackBishops, p.BlackRooks, p.BlackQueens, p.BlackKing
		ownPieces = p.BlackPieces
		color = types.ColorBlack
	}

	// Pawns.
	bb := pawns
	for bb != 0 {
		i := bitscan(bb)
		bb &= bb - 1
		p.MovePawn(color|types.Pawn, i, ml)
	}

	// Knights.
	bb = knights
	for bb != 0 {
		i := bitscan(bb)
		bb &= bb - 1
		p.MoveKnight(color|types.Knight, i, ml)
	}

	// Bishops.
	bb = bishops
	for bb != 0 {
		i := bitscan(bb)
		bb &= bb - 1
		p.MoveBishop(color|types.Bishop, i, ml)
	}

	// Rooks.
	bb = rooks
	for bb != 0 {
		i := bitscan(bb)
		bb &= bb - 1
		p.MoveRook(color|types.Rook, i, ml)
	}

	// Queens (rook + bishop attacks combined).
	bb = queens
	for bb != 0 {
		i := bitscan(bb)
		bb &= bb - 1
		p.MoveRook(color|types.Queen, i, ml)
		p.MoveBishop(color|types.Queen, i, ml)
	}

	// King.
	if king != 0 {
		i := bitscan(king)
		p.MoveKing(color|types.King, i, ml)
	}

	_ = ownPieces // used by the sub-generators internally
}