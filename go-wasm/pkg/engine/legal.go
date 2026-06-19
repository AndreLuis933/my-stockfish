package engine

import "webassemble/pkg/types"

// LegalMoves returns all pseudo-legal moves that do not leave the own king
// in check. It works by: generate pseudo-legal → for each, Make / test /
// Unmake. The make/unmake pattern is O(1) per move — no full board copy —
// which is critical for AI search performance.
//
// This handles pins, en-passant discovered checks, and king-moves-into-check
// automatically, because we test the actual resulting position.
func (p *Position) LegalMoves() []types.Move {
	pseudo := p.PseudoLegalMoves()
	moverColor := p.colorOfSide()
	legal := make([]types.Move, 0, len(pseudo))
	for _, m := range pseudo {
		p.Make(m)
		if !p.IsInCheck(moverColor) {
			legal = append(legal, m)
		}
		p.Unmake(m)
	}
	return legal
}