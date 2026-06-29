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

// Killer moves: quiet moves that caused a beta cutoff at a given ply in the
// current search. Two slots per ply — if a quiet move already cut off here,
// it's likely to do so again in sibling nodes (move ordering transitivity).
// Killer moves are not captures (those are ordered by MVV already).
//
// Stored as [maxPly][2]Move, indexed by ply. Cleared at the start of each new
// iterative-deepening root iteration (previous killers are stale at new depths).
//
// NOTE: must be sized by maxPly (256), not maxDepth (32) — the table is indexed
// by p.Ply() (= undoPly), which can exceed 32 during deep searches with check
// extensions. A [32] table panics in the browser on long searches.
type killerTable [maxPly][2]types.Move

// isKiller reports whether m is one of the two killer moves at the given ply.
func (k *killerTable) isKiller(ply int, m types.Move) bool {
	return (k[ply][0].From == m.From && k[ply][0].To == m.To && k[ply][0].Promotion == m.Promotion) ||
		(k[ply][1].From == m.From && k[ply][1].To == m.To && k[ply][1].Promotion == m.Promotion)
}

// storeKiller records a quiet cutoff move at the given ply. Slot 0 is replaced
// only if m is not already there; slot 1 is pushed down. Duplicates are
// avoided to keep both slots useful.
func (k *killerTable) storeKiller(ply int, m types.Move) {
	if k[ply][0].From == m.From && k[ply][0].To == m.To && k[ply][0].Promotion == m.Promotion {
		return
	}
	k[ply][1] = k[ply][0]
	k[ply][0] = m
}

// History heuristic: a [64][64] table counting how often each from→to move
// pair caused a cutoff in quiet positions. Used to order non-capture, non-
// killer moves. Higher count → searched earlier → more cutoffs.
//
// Indexed by from*64 + to for a flat array (4096 entries). Cleared at the
// start of each new iterative-deepening root iteration.
type historyTable [64 * 64]int

// scoreHistory returns the history score for a quiet move.
func (h *historyTable) score(m types.Move) int {
	return h[int(m.From)*64+int(m.To)]
}

// storeHistory increments the history score for a quiet cutoff move. The
// bonus is scaled by depth (deeper cutoffs are more valuable). The depth²
// scaling is the standard Stockfish-style formula.
func (h *historyTable) store(m types.Move, depth int) {
	if depth < 0 {
		depth = 0
	}
	h[int(m.From)*64+int(m.To)] += depth * depth
}

// ageHistory decays history scores between iterations by subtracting a
// constant from each entry, eventually zeroing stale moves. This is a
// simple form of aging — at each new root depth we want recent cutoffs
// to weigh more than old ones.
func (h *historyTable) age() {
	for i := range h {
		if h[i] > 8 {
			h[i] -= 8
		} else {
			h[i] = 0
		}
	}
}

// moveOrderScore assigns a heuristic score for ordering. The ranking, from
// highest to lowest:
//  1. previousBest / TT move (1 << 20) — from iterative deepening or TT probe
//  2. captures (1000 + MVV) — most valuable victim
//  3. killers (900) — quiet cutoffs at this ply
//  4. history (0..899) — quiet cutoffs across the tree
//  5. quiet moves (0)
//
// Better ordering → more alpha-beta cutoffs → exponentially fewer nodes.
func moveOrderScore(m types.Move, previousBest *types.Move, killers *killerTable, history *historyTable, ply int) int {
	if previousBest != nil &&
		m.From == previousBest.From &&
		m.To == previousBest.To &&
		m.Promotion == previousBest.Promotion {
		return 1 << 20
	}
	if m.Captured != 0 {
		return 1000 + engine.MaterialValue(m.Captured)
	}
	if killers != nil && killers.isKiller(ply, m) {
		return 900
	}
	if history != nil {
		return history.score(m)
	}
	return 0
}

// orderMoves sorts the move list in place by descending heuristic score.
// The previousBest move (from iterative deepening / TT) is forced to index 0
// by giving it the highest score — one sort pass, no second scan.
//
// Each move is scored exactly once into a parallel array, then sorted by the
// cached scores. Scoring once (not on every comparison) keeps the expensive
// moveOrderScore — killer/history lookups — out of the O(n²) comparison loop.
//
// Uses insertion sort: optimal for the small move lists (~20-40 moves) typical
// in chess. stdlib sort.Slice's introsort + closure overhead is slower here.
func orderMoves(ml *engine.MoveList, previousBest *types.Move, killers *killerTable, history *historyTable, ply int, scores *[256]int) {
	n := ml.Len()
	if n <= 1 {
		return
	}
	moves := ml.Slice()

	for i := range n {
		scores[i] = moveOrderScore(moves[i], previousBest, killers, history, ply)
	}

	for i := 1; i < n; i++ {
		si, mi := scores[i], moves[i]
		j := i
		for ; j > 0 && scores[j-1] < si; j-- {
			scores[j], moves[j] = scores[j-1], moves[j-1]
		}
		scores[j], moves[j] = si, mi
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