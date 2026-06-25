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

// nextGen increments the global generation counter and returns the new value.
// On wraparound (255 → 0), it clears the TT so that replacement priority
// comparisons stay monotonic within each epoch.
func nextGen(tt *engine.TranspositionTable) uint8 {
	genCounter++
	if genCounter == 0 {
		tt.Clear()
	}
	return genCounter
}

// trimSearchPosition bounds p's undo stack to the reversible-move window so
// the search gets a near-full ply budget regardless of game length. It mutates
// p directly (no copy): the search runs on the caller's Position and restores
// it via balanced Make/Unmake on every path, so the caller's state is unchanged
// when the search returns.
//
// The window is HalfmoveClock+1 — exactly the range IsRepetition scans (bounded
// by the halfmove clock in draw.go), so the search sees every draw-relevant
// position while discarding older irreversible history. With maxPly=512 and a
// max 101-ply window, the search has >=411 plies of headroom (maxDepth=64 plus
// check extensions and quiescence fit comfortably).
func trimSearchPosition(p *engine.Position) {
	p.TrimUndoStack(p.HalfmoveClock + 1)
}

// Search runs iterative deepening with a time budget. Returns the best move
// from the last fully-completed depth. Pass stopCh=nil for time-only abort;
// pass a closeable channel to allow external cancellation.
//
// A 32MB transposition table is created per call. For repeated searches on
// the same position tree (e.g. a full game), reuse a TT via SearchWithTT.
//
// The search runs directly on p (no copy): Make/Unmake are balanced on every
// path, so p is restored to its pre-search state when Search returns.
func Search(p *engine.Position, timeLimitMs int, stopCh <-chan struct{}) SearchResult {
	tt := engine.DefaultTranspositionTable()
	gen := nextGen(tt)
	ctx := &searchCtx{
		startTime:   nowMs(),
		timeLimitMs: float64(timeLimitMs),
		stopCh:      stopCh,
		tt:          tt,
		gen:         gen,
	}
	trimSearchPosition(p)
	return iterativeDeepening(p, ctx, maxDepth)
}

// SearchWithTT runs iterative deepening with a caller-provided transposition
// table. The TT is not cleared — entries accumulate across searches, which
// improves hit rates in games where positions recur. Call tt.Clear() between
// games if needed.
//
// The search runs directly on p (no copy): Make/Unmake are balanced on every
// path, so p is restored to its pre-search state when SearchWithTT returns.
func SearchWithTT(p *engine.Position, timeLimitMs int, stopCh <-chan struct{}, tt *engine.TranspositionTable) SearchResult {
	gen := nextGen(tt)
	ctx := &searchCtx{
		startTime:   nowMs(),
		timeLimitMs: float64(timeLimitMs),
		stopCh:      stopCh,
		tt:          tt,
		gen:         gen,
	}
	trimSearchPosition(p)
	return iterativeDeepening(p, ctx, maxDepth)
}

// SearchFixedDepth searches to the given depth. Uses iterative deepening
// internally so that if the search is aborted (e.g. by a UCI "stop" command),
// it can return the best move from the last completed depth instead of an
// empty result. Pass stopCh=nil to disable external abort.
//
// The search runs directly on p (no copy): Make/Unmake are balanced on every
// path, so p is restored to its pre-search state when SearchFixedDepth returns.
func SearchFixedDepth(p *engine.Position, depth int, stopCh <-chan struct{}) SearchResult {
	tt := engine.DefaultTranspositionTable()
	gen := nextGen(tt)
	ctx := &searchCtx{
		startTime:   nowMs(),
		timeLimitMs: 1e18,
		stopCh:      stopCh,
		tt:          tt,
		gen:         gen,
	}
	trimSearchPosition(p)
	return iterativeDeepening(p, ctx, depth)
}

// SearchFixedDepthWithTT is the caller-provided-TT variant of SearchFixedDepth.
//
// The search runs directly on p (no copy): Make/Unmake are balanced on every
// path, so p is restored to its pre-search state when SearchFixedDepthWithTT
// returns.
func SearchFixedDepthWithTT(p *engine.Position, depth int, stopCh <-chan struct{}, tt *engine.TranspositionTable) SearchResult {
	gen := nextGen(tt)
	ctx := &searchCtx{
		startTime:   nowMs(),
		timeLimitMs: 1e18,
		stopCh:      stopCh,
		tt:          tt,
		gen:         gen,
	}
	trimSearchPosition(p)
	return iterativeDeepening(p, ctx, depth)
}