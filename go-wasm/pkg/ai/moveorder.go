package ai

import (
	"webassemble/pkg/engine"
	"webassemble/pkg/types"
)

// sideColor returns the color of the side to move.
func sideColor(p *engine.Position) types.Piece {
	if p.WhiteToMove {
		return types.ColorWhite
	}
	return types.ColorBlack
}

// moveOrderScore assigns a heuristic score for ordering: the previous-best
// move gets the highest score (so it lands at index 0), captures next (weighted
// by captured piece value), quiet moves get 0. Better ordering → more alpha-beta
// cutoffs → exponentially fewer nodes searched.
func moveOrderScore(m types.Move, previousBest *types.Move) int {
	if previousBest != nil &&
		m.From == previousBest.From &&
		m.To == previousBest.To &&
		m.Promotion == previousBest.Promotion {
		return 1 << 20
	}
	if m.Captured != 0 {
		return 1000 + engine.MaterialValue(m.Captured)
	}
	return 0
}

// orderMoves sorts the move list in place by descending heuristic score.
// The previousBest move (from iterative deepening) is forced to index 0 by
// giving it the highest score — one sort pass, no second scan.
//
// Uses insertion sort: optimal for the small move lists (~20-40 moves) typical
// in chess. stdlib sort.Slice's introsort + closure overhead is slower here.
func orderMoves(ml *engine.MoveList, previousBest *types.Move) {
	n := ml.Len()
	if n <= 1 {
		return
	}
	moves := ml.Slice()

	for i := 1; i < n; i++ {
		si := moveOrderScore(moves[i], previousBest)
		for j := i; j > 0; j-- {
			sj := moveOrderScore(moves[j-1], previousBest)
			if si <= sj {
				break
			}
			moves[j], moves[j-1] = moves[j-1], moves[j]
		}
	}
}

// noLegalMoveScore returns the terminal score when the side to move has no legal
// moves: checkmate (negative, scaled by ply so closer mates are preferred) or
// stalemate (draw, score 0).
//
// Uses ply (distance from root) not depth: a mate found at ply N is scored
// -winScore + N, so mates closer to the root (smaller N) have higher scores
// and are preferred. Using depth would be wrong because the check extension
// modifies depth, and depth doesn't reflect the actual distance from root.
func noLegalMoveScore(inCheck bool, ply int) int {
	if inCheck {
		return -winScore + ply
	}
	return 0
}