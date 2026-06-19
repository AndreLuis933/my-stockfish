package engine

import "webassemble/pkg/types"

// PseudoLegalMoves generates all moves that are legal *ignoring* whether
// they leave the own king in check. The filtering happens in LegalMoves.
//
// Writes into the caller-owned MoveList — zero heap allocation.
func (p *Position) PseudoLegalMoves(ml *MoveList) {
	ml.Clear()
	for i, piece := range p.Board {
		if piece == 0 || piece.IsWhite() != p.WhiteToMove {
			continue
		}

		switch piece & types.TypeMask {
		case types.Pawn:
			p.MovePawn(piece, i, ml)
		case types.Rook:
			p.MoveRook(piece, i, ml)
		case types.Bishop:
			p.MoveBishop(piece, i, ml)
		case types.Queen:
			p.MoveRook(piece, i, ml)
			p.MoveBishop(piece, i, ml)
		case types.King:
			p.MoveKing(piece, i, ml)
		case types.Knight:
			p.MoveKnight(piece, i, ml)
		}
	}
}