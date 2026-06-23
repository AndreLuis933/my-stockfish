package ai

import (
	"webassemble/pkg/engine"
	"webassemble/pkg/types"
)

const (
	winScore      = 30_000
	nodeCheckMask = 2047
	maxDepth      = 32
	maxPly        = 256
	negInf        = -32_000
)

// SearchResult holds the outcome of an iterative deepening search.
type SearchResult struct {
	Move   types.Move
	Score  int
	Depth  int
	Nodes  int
	TimeMs int64
}

// searchCtx tracks time, node count, abort state, TT, and move-ordering
// heuristics (killers + history) across the recursion. The killer and history
// tables are per-search (cleared at the start of each iterative-deepening
// iteration by the root caller).
type searchCtx struct {
	startTime   float64
	timeLimitMs float64
	nodes       int
	aborted     bool
	stopCh      <-chan struct{}
	tt          *engine.TranspositionTable
	killers     killerTable
	history     historyTable
	// disableNullMove turns off null-move pruning. Used by tests to A/B
	// compare pruning behavior and by positions where pruning is known to
	// be unsafe. A single bool checked once per node — negligible cost.
	disableNullMove bool
}

// shouldStop checks if the search has exceeded its time budget or received an
// external stop signal. Called every nodeCheckMask nodes to amortize the cost.
func (c *searchCtx) shouldStop() {
	if c.nodes&nodeCheckMask != 0 {
		return
	}
	if nowMs()-c.startTime >= c.timeLimitMs {
		c.aborted = true
		return
	}
	if c.stopCh != nil {
		select {
		case <-c.stopCh:
			c.aborted = true
		default:
		}
	}
}