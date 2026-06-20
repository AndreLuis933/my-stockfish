package ai

import (
	"webassemble/pkg/engine"
	"webassemble/pkg/types"
)

// negamax is the recursive search core. Returns the score from the perspective
// of the side to move and the best move found, using alpha-beta pruning with
// negamax symmetry.
//
// At internal nodes the caller ignores the best move (only the score matters
// for pruning). At the root the caller uses it. This avoids duplicating the
// move loop in a separate root function.
//
// previousBest (from iterative deepening) is used for move ordering at the
// root only — pass nil at internal nodes.
//
// Lazy legality: pseudo-legal moves are generated, Make is applied, and the
// move is skipped if the side that just moved is now in check — one
// Make/Unmake per move (the minimum possible).
func negamax(p *engine.Position, depth, alpha, beta int, ctx *searchCtx, previousBest *types.Move) (int, types.Move) {
	ctx.nodes++
	ctx.shouldStop()
	if ctx.aborted {
		return 0, types.Move{}
	}

	if p.HalfmoveClock >= 100 {
		return 0, types.Move{}
	}
	if depth == 0 {
		return Evaluate(p), types.Move{}
	}

	var ml engine.MoveList
	p.PseudoLegalMoves(&ml)
	orderMoves(&ml, previousBest)

	moverColor := sideColor(p)
	inCheck := p.IsInCheck(moverColor)
	best := negInf
	bestMove := types.Move{}

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
		score, _ := negamax(p, depth-1, -beta, -alpha, ctx, nil)
		score = -score
		p.Unmake(m)

		if ctx.aborted {
			break
		}
		if score > best {
			best = score
			bestMove = m
		}
		if best > alpha {
			alpha = best
		}
		if alpha >= beta {
			break
		}
	}

	if best == negInf {
		return noLegalMoveScore(inCheck, depth), types.Move{}
	}
	return best, bestMove
}