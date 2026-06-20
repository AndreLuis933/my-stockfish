package ai

import (
	"webassemble/pkg/engine"
	"webassemble/pkg/types"
)

// SearchResult holds the outcome of an iterative deepening search.
type SearchResult struct {
	Move   types.Move
	Score  int
	Depth  int
	Nodes  int
	TimeMs int64
}

// searchCtx tracks time, node count, and abort state across the recursion.
type searchCtx struct {
	startTime   float64
	timeLimitMs float64
	nodes       int
	aborted     bool
}

const nodeCheckMask = 2047

func sideColor(p *engine.Position) types.Piece {
	if p.WhiteToMove {
		return types.ColorWhite
	}
	return types.ColorBlack
}

// orderMoves sorts captures before quiet moves, then places previousBest
// first (for iterative deepening — searching the best move from the previous
// depth first improves alpha-beta pruning).
func orderMoves(ml *engine.MoveList, previousBest *types.Move) {
	n := ml.Len()
	if n <= 1 {
		return
	}
	moves := ml.Slice()

	score := func(m types.Move) int {
		if m.Captured != 0 {
			return 1000 + int(m.Captured&types.TypeMask)
		}
		return 0
	}

	for i := 1; i < n; i++ {
		for j := i; j > 0 && score(moves[j]) > score(moves[j-1]); j-- {
			moves[j], moves[j-1] = moves[j-1], moves[j]
		}
	}

	if previousBest != nil {
		for i := 0; i < n; i++ {
			if moves[i].From == previousBest.From && moves[i].To == previousBest.To &&
				moves[i].Promotion == previousBest.Promotion {
				if i > 0 {
					best := moves[i]
					copy(moves[1:i+1], moves[:i])
					moves[0] = best
				}
				break
			}
		}
	}
}

// negamax is the recursive search core. Returns the score from the
// perspective of the side to move. Uses alpha-beta pruning.
//
// Lazy legality: pseudo-legal moves are generated, Make is applied, and the
// move is skipped if the side that just moved is now in check. This costs one
// Make/Unmake per move (the minimum possible) — cheaper than calling
// LegalMoves which would Make/Unmake a second time.
func negamax(p *engine.Position, depth, alpha, beta int, ctx *searchCtx) int {
	ctx.nodes++
	if (ctx.nodes&nodeCheckMask) == 0 && nowMs()-ctx.startTime >= ctx.timeLimitMs {
		ctx.aborted = true
	}
	if ctx.aborted {
		return 0
	}

	status := p.CurrentStatus()
	if status.IsGameOver() {
		if status == engine.StatusDraw {
			return 0
		}
		return -winScore + depth
	}
	if depth == 0 {
		return Evaluate(p)
	}

	var ml engine.MoveList
	p.PseudoLegalMoves(&ml)
	orderMoves(&ml, nil)

	moverColor := sideColor(p)
	best := -1 << 30

	for i := 0; i < ml.Len(); i++ {
		if ctx.aborted {
			break
		}
		m := ml.Get(i)
		p.Make(m)
		if p.IsInCheck(moverColor) {
			p.Unmake(m)
			continue
		}
		score := -negamax(p, depth-1, -beta, -alpha, ctx)
		p.Unmake(m)

		if score > best {
			best = score
		}
		if best > alpha {
			alpha = best
		}
		if alpha >= beta {
			break
		}
	}

	if best == -1<<30 {
		return -winScore + depth
	}
	return best
}

// searchAtDepth searches all root moves at a fixed depth. Returns the best
// move, its score, and whether the search completed without aborting.
func searchAtDepth(p *engine.Position, depth int, ctx *searchCtx, previousBest *types.Move) (types.Move, int, bool) {
	var ml engine.MoveList
	p.PseudoLegalMoves(&ml)
	orderMoves(&ml, previousBest)

	moverColor := sideColor(p)
	bestScore := -1 << 30
	bestMove := types.Move{}

	for i := 0; i < ml.Len(); i++ {
		if ctx.aborted {
			return types.Move{}, 0, false
		}
		m := ml.Get(i)
		p.Make(m)
		if p.IsInCheck(moverColor) {
			p.Unmake(m)
			continue
		}
		score := -negamax(p, depth-1, -1<<30, 1<<30, ctx)
		p.Unmake(m)

		if !ctx.aborted && score > bestScore {
			bestScore = score
			bestMove = m
		}
	}

	return bestMove, bestScore, !ctx.aborted
}

// Search runs iterative deepening: depth 1, 2, 3... until the time budget
// expires. Returns the best move from the last fully-completed depth.
// Partial results from an aborted depth are discarded (unreliable).
func Search(p *engine.Position, timeLimitMs int) SearchResult {
	ctx := &searchCtx{
		startTime:   nowMs(),
		timeLimitMs: float64(timeLimitMs),
	}

	var best SearchResult
	var previousBest *types.Move

	for depth := 1; depth <= 32; depth++ {
		move, score, complete := searchAtDepth(p, depth, ctx, previousBest)
		if !complete {
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

// SearchFixedDepth runs a fixed-depth search (no time limit). Useful for
// benchmarking and testing — guarantees the search reaches the given depth.
func SearchFixedDepth(p *engine.Position, depth int) SearchResult {
	ctx := &searchCtx{
		startTime:   nowMs(),
		timeLimitMs: 1e18,
	}

	var best SearchResult
	var previousBest *types.Move

	for d := 1; d <= depth; d++ {
		move, score, complete := searchAtDepth(p, d, ctx, previousBest)
		if !complete {
			break
		}
		previousBest = &move
		best = SearchResult{
			Move:  move,
			Score: score,
			Depth: d,
			Nodes: ctx.nodes,
		}
		if score >= winScore || score <= -winScore {
			break
		}
	}

	best.TimeMs = int64(nowMs() - ctx.startTime)
	return best
}