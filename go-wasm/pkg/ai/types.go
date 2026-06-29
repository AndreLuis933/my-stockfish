package ai

import (
	"webassemble/pkg/engine"
	"webassemble/pkg/types"
)

const (
	winScore      = 30_000
	nodeCheckMask = 2047
	maxDepth      = 64
	maxPly        = 512
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

// SearchLineResult holds one line of a Multi-PV search: the principal
// variation (as a sequence of moves), its score from the side-to-move's
// perspective, the depth reached, and search statistics. Lines are sorted
// by score descending (best line first).
type SearchLineResult struct {
	Moves  []types.Move
	Score  int
	Depth  int
	Nodes  int
	TimeMs int64
}

// genCounter is a package-level search generation counter, incremented before
// each Search/SearchFixedDepth call. It is passed to TT.Store so that
// replacement priority (gen + depth) reflects recency: old deep entries from
// early moves decay as the game progresses. On wraparound (255 → 0), the
// caller clears the TT to keep comparisons monotonic within each epoch.
var genCounter uint8

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
	gen         uint8
	killers     killerTable
	history     historyTable
	// orderScratch is a reusable score buffer for orderMoves. Kept on the ctx
	// so it's allocated once per search instead of being a stack array zeroed
	// on every node — zeroing 2KB per node was ~5% of total search time.
	// orderMoves writes [0:n] before reading, so stale values never leak.
	orderScratch [256]int
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