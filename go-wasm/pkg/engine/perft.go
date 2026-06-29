package engine

// Perft (performance test) counts the number of leaf nodes at the given depth.
// It is the standard way to validate move generation: the node counts for
// known positions are published, so we can compare.
//
// Reference results (must stay identical after refactors):
//   depth 1 = 20, depth 2 = 400, depth 3 = 8902, depth 4 = 197281,
//   depth 5 = 4865609  (from the starting position).
//
// Uses PseudoLegalMoves + inline Make/IsInCheck/Unmake (same pattern as the AI
// search). This avoids the double Make/Unmake overhead of calling LegalMoves.
func (p *Position) Perft(depth int) int {
	if depth == 0 {
		return 1
	}

	var ml MoveList
	p.PseudoLegalMoves(&ml)
	moverColor := p.colorOfSide()
	nodes := 0
	for i := 0; i < ml.n; i++ {
		m := ml.moves[i]
		p.Make(m)
		if !p.IsInCheck(moverColor) {
			nodes += p.Perft(depth - 1)
		}
		p.Unmake(m)
	}

	return nodes
}