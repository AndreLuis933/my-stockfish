package ai

import (
	"webassemble/pkg/engine"
	"webassemble/pkg/types"
)

// iterativeDeepening runs depth 1, 2, 3... up to depthCap, returning the best
// move from the last fully-completed depth. Aborted partial results are
// discarded. Stops early on a forced mate.
//
// Used by time-limited search, where the max reachable depth is unknown — if
// the time runs out mid-depth, we fall back to the best move from the previous
// depth. The previous depth's best move is searched first at each new depth to
// maximize alpha-beta cutoffs.
func iterativeDeepening(p *engine.Position, ctx *searchCtx, depthCap int) SearchResult {
	var best SearchResult
	var previousBest *types.Move

	for depth := 1; depth <= depthCap; depth++ {
		score, move := negamax(p, depth, negInf, -negInf, ctx, previousBest)
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
func Search(p *engine.Position, timeLimitMs int, stopCh <-chan struct{}) SearchResult {
	ctx := &searchCtx{
		startTime:   nowMs(),
		timeLimitMs: float64(timeLimitMs),
		stopCh:      stopCh,
	}
	return iterativeDeepening(p, ctx, maxDepth)
}

// SearchFixedDepth searches directly to the given depth — no iterative
// deepening. This is faster than ID when the target depth is known in advance
// (benchmarks, tests, depth-limited play) because it skips the shallower
// passes. Pass stopCh=nil to disable external abort.
func SearchFixedDepth(p *engine.Position, depth int, stopCh <-chan struct{}) SearchResult {
	ctx := &searchCtx{
		startTime:   nowMs(),
		timeLimitMs: 1e18,
		stopCh:      stopCh,
	}

	score, move := negamax(p, depth, negInf, -negInf, ctx, nil)
	if ctx.aborted {
		return SearchResult{}
	}

	return SearchResult{
		Move:   move,
		Score:  score,
		Depth:  depth,
		Nodes:  ctx.nodes,
		TimeMs: int64(nowMs() - ctx.startTime),
	}
}