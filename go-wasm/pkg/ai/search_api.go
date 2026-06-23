package ai

import (
	"webassemble/pkg/engine"
	"webassemble/pkg/types"
)

// aspirationWindow is the half-width of the aspiration window around the
// previous iteration's score. If the score falls outside [score-window,
// score+window], we re-search with a full window.
const aspirationWindow = 50

// iterativeDeepening runs depth 1, 2, 3... up to depthCap, returning the best
// move from the last fully-completed depth. Aborted partial results are
// discarded. Stops early on a forced mate.
//
// Used by time-limited search, where the max reachable depth is unknown — if
// the time runs out mid-depth, we fall back to the best move from the previous
// depth. The previous depth's best move is searched first at each new depth to
// maximize alpha-beta cutoffs. The TT accumulates across all depths.
//
// Aspiration windows: from depth 2 onward, we search with a narrow window
// around the previous iteration's score instead of [-inf, +inf]. When the
// score falls inside the window (the common case), this prunes much more. On
// fail-low or fail-high, we re-search with the full window — the TT makes
// this re-search cheap (subtree nodes are already cached).
func iterativeDeepening(p *engine.Position, ctx *searchCtx, depthCap int) SearchResult {
	var best SearchResult
	var previousBest *types.Move

	for depth := 1; depth <= depthCap; depth++ {
		// Age the history table between iterations: older cutoffs become
		// less relevant as the search deepens. Killers are overwritten
		// naturally as new cutoffs replace old ones.
		ctx.history.age()

		var score int
		var move types.Move

		if depth >= 3 && previousBest != nil {
			// Aspiration window around the previous iteration's score.
			alpha := best.Score - aspirationWindow
			beta := best.Score + aspirationWindow

			for {
				score, move = negamax(p, depth, alpha, beta, ctx, previousBest)
				if ctx.aborted {
					break
				}

				if score <= alpha {
					// Fail-low: score is an upper bound, true score is
					// lower. Widen the window downward and re-search.
					alpha = negInf
				} else if score >= beta {
					// Fail-high: score is a lower bound, true score is
					// higher. Widen upward and re-search.
					beta = -negInf
				} else {
					// Score is exact — inside the window.
					break
				}
			}
		} else {
			score, move = negamax(p, depth, negInf, -negInf, ctx, previousBest)
		}

		if ctx.aborted {
			break
		}
		previousBest = &move
		best = SearchResult{
			Move:   move,
			Score:  score,
			Depth:  depth,
			Nodes:  ctx.nodes,
			TimeMs: int64(nowMs() - ctx.startTime),
		}
		if score >= winScore || score <= -winScore {
			break
		}
	}

	return best
}

// Search runs iterative deepening with a time budget. Returns the best move
// from the last fully-completed depth. Pass stopCh=nil for time-only abort;
// pass a closeable channel to allow external cancellation.
//
// A 16MB transposition table is created per call. For repeated searches on
// the same position tree (e.g. a full game), reuse a TT via SearchWithTT.
func Search(p *engine.Position, timeLimitMs int, stopCh <-chan struct{}) SearchResult {
	tt := engine.DefaultTranspositionTable()
	ctx := &searchCtx{
		startTime:   nowMs(),
		timeLimitMs: float64(timeLimitMs),
		stopCh:      stopCh,
		tt:          tt,
	}
	return iterativeDeepening(p, ctx, maxDepth)
}

// SearchWithTT runs iterative deepening with a caller-provided transposition
// table. The TT is not cleared — entries accumulate across searches, which
// improves hit rates in games where positions recur. Call tt.Clear() between
// games if needed.
func SearchWithTT(p *engine.Position, timeLimitMs int, stopCh <-chan struct{}, tt *engine.TranspositionTable) SearchResult {
	ctx := &searchCtx{
		startTime:   nowMs(),
		timeLimitMs: float64(timeLimitMs),
		stopCh:      stopCh,
		tt:          tt,
	}
	return iterativeDeepening(p, ctx, maxDepth)
}

// SearchFixedDepth searches to the given depth. Uses iterative deepening
// internally so that if the search is aborted (e.g. by a UCI "stop" command),
// it can return the best move from the last completed depth instead of an
// empty result. Pass stopCh=nil to disable external abort.
func SearchFixedDepth(p *engine.Position, depth int, stopCh <-chan struct{}) SearchResult {
	tt := engine.DefaultTranspositionTable()
	ctx := &searchCtx{
		startTime:   nowMs(),
		timeLimitMs: 1e18,
		stopCh:      stopCh,
		tt:          tt,
	}
	return iterativeDeepening(p, ctx, depth)
}

// SearchFixedDepthWithTT is the caller-provided-TT variant of SearchFixedDepth.
func SearchFixedDepthWithTT(p *engine.Position, depth int, stopCh <-chan struct{}, tt *engine.TranspositionTable) SearchResult {
	ctx := &searchCtx{
		startTime:   nowMs(),
		timeLimitMs: 1e18,
		stopCh:      stopCh,
		tt:          tt,
	}
	return iterativeDeepening(p, ctx, depth)
}