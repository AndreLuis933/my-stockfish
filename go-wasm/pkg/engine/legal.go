package engine

import "webassemble/pkg/types"

// LegalMoves returns all pseudo-legal moves that do not leave the own king
// in check. It works by: generate pseudo-legal → for each, Make / test /
// Unmake. The make/unmake pattern is O(1) per move — no full board copy —
// which is critical for AI search performance.
//
// This handles pins, en-passant discovered checks, and king-moves-into-check
// automatically, because we test the actual resulting position.
//
// Writes into the caller-owned MoveList — zero heap allocation.
func (p *Position) LegalMoves(ml *MoveList) {
	var pseudo MoveList
	p.PseudoLegalMoves(&pseudo)
	ml.Clear()
	moverColor := p.colorOfSide()
	for i := 0; i < pseudo.n; i++ {
		m := pseudo.moves[i]
		p.Make(m)
		if !p.IsInCheck(moverColor) {
			ml.Add(m)
		}
		p.Unmake(m)
	}
}

// LegalMovesSlice is a convenience wrapper for callers that need a []Move
// (e.g., JSON marshaling in the WASM bridge). The MoveList escapes to the
// heap here — use LegalMoves directly in hot paths.
func (p *Position) LegalMovesSlice() []types.Move {
	var ml MoveList
	p.LegalMoves(&ml)
	return ml.Slice()
}