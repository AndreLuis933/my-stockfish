package ai

import (
	"webassemble/pkg/engine"
)

// quiescence is the "stand-pat" search that extends past the depth horizon
// with only captures (and promotions). It prevents the horizon effect: without
// it, the engine evaluates a position at depth 0 even if a queen is hanging
// one ply deeper, leading to grossly wrong evaluations and bad moves.
//
// Algorithm:
//  1. Stand-pat: evaluate the current position. If it's already >= beta,
//     the opponent won't allow this position — return beta (cutoff).
//  2. Try each capture move. If any raises the score above alpha, update.
//  3. If no capture improves alpha, the stand-pat score stands.
//
// This is the classic "fail-hard" quiescence with stand-pat. It does NOT
// search check evasions — if the side to move is in check, the stand-pat
// is unreliable, but the main search will have handled that by not cutting
// at depth 0 when in check (the move loop finds evasions).
//
// Mate scores are NOT adjusted for ply here — quiescence doesn't detect
// mate (it only searches captures, so it can't know if a position is mate).
func quiescence(p *engine.Position, alpha, beta int, ctx *searchCtx) int {
	ctx.nodes++
	ctx.shouldStop()
	if ctx.aborted {
		return 0
	}

	// Hard ply limit: quiescence has no depth parameter, so without this
	// guard a long capture chain can recurse past maxPly and overflow the
	// undo stack in Make. Return the static eval as a safe fallback.
	if p.Ply() >= maxPly {
		return Evaluate(p)
	}

	// Threefold repetition: return draw before stand-pat, so a repeating
	// position returns 0 instead of a stale winning eval. The 50-move rule
	// is not checked here because quiescence doesn't detect mate anyway.
	if p.IsRepetition() {
		return 0
	}

	// Stand-pat: the side to move can "pass" and keep the current position.
	// If the static eval is already good enough, prune.
	standPat := Evaluate(p)
	if standPat >= beta {
		return standPat
	}
	if standPat > alpha {
		alpha = standPat
	}

	// 50-move rule: draw in quiescence too.
	if p.HalfmoveClock >= 100 {
		return 0
	}

	var ml engine.MoveList
	p.PseudoLegalCaptures(&ml)
	orderMoves(&ml, nil, nil, nil, 0, &ctx.orderScratch)

	moverColor := sideColor(p)

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
		score := -quiescence(p, -beta, -alpha, ctx)
		p.Unmake(m)

		if ctx.aborted {
			break
		}
		if score >= beta {
			return score
		}
		if score > alpha {
			alpha = score
		}
	}

	return alpha
}