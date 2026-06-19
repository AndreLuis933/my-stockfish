package engine

import "webassemble/pkg/types"

// positionSnapshot is the saved state used by MakeMove/restore during legal
// move filtering and Perft. It is a full copy of the mutable fields — simple
// but not the fastest. Will be replaced by Make/Unmake in a later refactor
// (Step 3 of the plan), once moves carry their own flags.
type positionSnapshot struct {
	board            types.Board
	enPassantCapture int
	enPassantTarget  int
	whiteToMove      bool
	castlingRights   types.CastlingRights
}

// snapshot copies the current mutable state so it can be restored after a
// trial move (used by LegalMoves and Perft).
func (p *Position) snapshot() positionSnapshot {
	return positionSnapshot{
		board:            p.Board,
		enPassantCapture: p.EnPassantCapture,
		enPassantTarget:  p.EnPassantTarget,
		whiteToMove:      p.WhiteToMove,
		castlingRights:   p.CastlingRights,
	}
}

// restore puts a previously saved snapshot back into the position.
func (p *Position) restore(s positionSnapshot) {
	p.Board = s.board
	p.EnPassantCapture = s.enPassantCapture
	p.EnPassantTarget = s.enPassantTarget
	p.WhiteToMove = s.whiteToMove
	p.CastlingRights = s.castlingRights
}

// LegalMoves returns all pseudo-legal moves that do not leave the own king
// in check. It works by: generate pseudo-legal → for each, snapshot / make /
// test / restore. This handles pins, en-passant discovered checks, and
// king-moves-into-check automatically.
func (p *Position) LegalMoves() []types.Move {
	pseudo := p.PseudoLegalMoves()
	moverColor := p.colorOfSide()
	legal := make([]types.Move, 0, len(pseudo))
	for _, m := range pseudo {
		saved := p.snapshot()
		p.Make(m)
		if !p.IsInCheck(moverColor) {
			legal = append(legal, m)
		}
		p.restore(saved)
	}
	return legal
}