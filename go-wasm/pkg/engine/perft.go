package engine

// Perft (performance test) counts the number of leaf nodes at the given depth.
// It is the standard way to validate move generation: the node counts for
// known positions are published, so we can compare.
//
// Reference results (must stay identical after refactors):
//   depth 1 = 20, depth 2 = 400, depth 3 = 8902, depth 4 = 197281,
//   depth 5 = 4865609  (from the starting position).
//
// Uses Make/Unmake instead of full board copies — the make/unmake pattern
// is the performance foundation for the AI search.
func (p *Position) Perft(depth int) int {
	if depth == 0 {
		return 1
	}

	moves := p.LegalMoves()
	nodes := 0

	for _, move := range moves {
		p.Make(move)
		nodes += p.Perft(depth - 1)
		p.Unmake(move)
	}

	return nodes
}