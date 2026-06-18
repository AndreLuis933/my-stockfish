package engine

import "webassemble/pkg/types"

// PseudoLegalMoves generates all moves that are legal *ignoring* whether
// they leave the own king in check. The filtering happens in LegalMoves.
//
// This is a method on *Position because it reads p.Board and p.WhiteToMove.
// The receiver name `p` is the Go convention (short, consistent across files).
func (p *Position) PseudoLegalMoves() []types.Move {
	var moves []types.Move
	for i, piece := range p.Board {
		if piece == 0 || piece.IsWhite() != p.WhiteToMove {
			continue
		}

		switch piece & types.TypeMask {
		case types.Pawn:
			moves = p.MovePawn(piece, i, moves)
		case types.Rook:
			moves = p.MoveRook(piece, i, moves)
		case types.Bishop:
			moves = p.MoveBishop(piece, i, moves)
		case types.Queen:
			moves = p.MoveRook(piece, i, moves)
			moves = p.MoveBishop(piece, i, moves)
		case types.King:
			moves = p.MoveKing(piece, i, moves)
		case types.Knight:
			moves = p.MoveKnight(piece, i, moves)
		}
	}
	return moves
}