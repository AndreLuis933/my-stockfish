package ai

import (
	"sort"

	"webassemble/pkg/engine"
	"webassemble/pkg/types"
)

// pvMaxLength bounds the reconstructed principal variation length. A 64-ply PV
// is more than enough for display; deeper lines are rarely useful and the walk
// is bounded to avoid pathological cases.
const pvMaxLength = 64

// SearchMultiPV runs iterative deepening and returns numLines distinct
// principal variations, sorted by score descending (best line first). Each
// line includes its PV move list, score (from the side-to-move perspective),
// depth, and search statistics.
//
// Implementation: at each depth, the root move list is searched line by line.
// After a line is searched, its best root move is excluded from subsequent
// lines' searches at this depth (root-level move exclusion). The TT accumulates
// across all lines and depths, so later lines and deeper iterations benefit
// from earlier work.
//
// The search runs directly on p (no copy): Make/Unmake are balanced on every
// path, so p is restored to its pre-search state when SearchMultiPV returns.
func SearchMultiPV(p *engine.Position, timeLimitMs int, numLines int, stopCh <-chan struct{}, tt *engine.TranspositionTable) []SearchLineResult {
	if numLines < 1 {
		numLines = 1
	}
	gen := nextGen(tt)
	ctx := &searchCtx{
		startTime:   nowMs(),
		timeLimitMs: float64(timeLimitMs),
		stopCh:      stopCh,
		tt:          tt,
		gen:         gen,
	}
	trimSearchPosition(p)

	var ml engine.MoveList
	p.PseudoLegalMoves(&ml)

	legalMoves := make([]types.Move, 0, ml.Len())
	moverColor := sideColor(p)
	for i := 0; i < ml.Len(); i++ {
		m := ml.Get(i)
		p.Make(m)
		if p.IsInCheck(moverColor) {
			p.Unmake(m)
			continue
		}
		p.Unmake(m)
		legalMoves = append(legalMoves, m)
	}

	if numLines > len(legalMoves) {
		numLines = len(legalMoves)
	}
	if numLines == 0 {
		return nil
	}

	var bestLines []SearchLineResult

	for depth := 1; depth <= maxDepth; depth++ {
		ctx.history.age()

		lines := make([]SearchLineResult, 0, numLines)
		excluded := make([]types.Move, 0, numLines)
		depthNodesStart := ctx.nodes

		for lineIdx := 0; lineIdx < numLines; lineIdx++ {
			if ctx.aborted {
				break
			}

			score, move := searchRootExcluding(p, depth, ctx, legalMoves, excluded)
			if ctx.aborted {
				break
			}

			pv := principalVariation(p, tt, pvMaxLength)
			if len(pv) == 0 {
				pv = []types.Move{move}
			}

			lines = append(lines, SearchLineResult{
				Moves:  pv,
				Score:  score,
				Depth:  depth,
				Nodes:  ctx.nodes - depthNodesStart,
				TimeMs: int64(nowMs() - ctx.startTime),
			})

			excluded = append(excluded, move)
		}

		if ctx.aborted && len(lines) < numLines {
			if len(lines) > 0 && bestLines == nil {
				sort.SliceStable(lines, func(i, j int) bool {
					return lines[i].Score > lines[j].Score
				})
				bestLines = lines
			}
			break
		}
		if len(lines) == numLines {
			sort.SliceStable(lines, func(i, j int) bool {
				return lines[i].Score > lines[j].Score
			})
			bestLines = lines
		}

		allMate := true
		for _, l := range lines {
			if l.Score < winScore-1000 && l.Score > -(winScore - 1000) {
				allMate = false
				break
			}
		}
		if allMate {
			break
		}
	}

	return bestLines
}

// searchRootExcluding searches the root position at the given depth, skipping
// any root move matching an entry in excluded. Returns the best score
// (side-to-move perspective) and its move.
//
// excluded holds the best moves already found for other PV lines at this depth;
// excluding them forces the search to find the next-best line.
//
// This is a full-window root search (not null-window): every non-excluded
// legal move is searched at [-inf, +inf] for the first move, then with a
// null window [-best, best] for subsequent moves (PV-search style). This
// guarantees the maximum-scoring move is found, not just the first to beat
// alpha.
func searchRootExcluding(p *engine.Position, depth int, ctx *searchCtx, legalMoves []types.Move, excluded []types.Move) (int, types.Move) {
	best := negInf
	bestMove := types.Move{}
	moverColor := sideColor(p)
	first := true

	for _, m := range legalMoves {
		if ctx.aborted {
			break
		}
		if isExcluded(m, excluded) {
			continue
		}

		p.Make(m)
		if p.IsInCheck(moverColor) {
			p.Unmake(m)
			continue
		}

		var score int
		if first {
			score, _ = negamax(p, depth-1, negInf, -negInf, ctx, nil)
			first = false
		} else {
			score, _ = negamax(p, depth-1, -best-1, -best, ctx, nil)
			if score > best && !ctx.aborted {
				score, _ = negamax(p, depth-1, negInf, -best, ctx, nil)
			}
		}
		score = -score
		p.Unmake(m)

		if ctx.aborted {
			break
		}
		if score > best {
			best = score
			bestMove = m
		}
	}

	if best == negInf {
		return 0, types.Move{}
	}
	return best, bestMove
}

// isExcluded reports whether m matches any move in the excluded list (by
// from/to/promotion). Promotion color is ignored — only the piece type matters.
func isExcluded(m types.Move, excluded []types.Move) bool {
	for _, e := range excluded {
		if m.From == e.From && m.To == e.To && (m.Promotion&types.TypeMask) == (e.Promotion&types.TypeMask) {
			return true
		}
	}
	return false
}