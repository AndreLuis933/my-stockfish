package ai

import (
	"webassemble/pkg/engine"
	"webassemble/pkg/types"
)

// TT flags: how the stored score relates to the true value.
const (
	ttExact = 1 // score is the exact value
	ttLower = 2 // score is a lower bound (beta cutoff)
	ttUpper = 3 // score is an upper bound (no move improved alpha)
)

// mateScore returns true if the score is near a forced mate (win or loss).
// Used to adjust mate scores for TT storage: mate distances are relative to
// the ply where they're found, so they must be stored as absolute values.
func mateScore(score int) bool {
	return score >= winScore-1000 || score <= -(winScore - 1000)
}

// scoreToTT converts a search score to TT storage form. Mate scores are
// adjusted by ply so they're stored as absolute distances from the root.
func scoreToTT(score, ply int) int16 {
	if score >= winScore-1000 {
		return int16(score + ply)
	}
	if score <= -(winScore - 1000) {
		return int16(score - ply)
	}
	return int16(score)
}

// scoreFromTT converts a TT score back to search form. Reverses the ply
// adjustment so the score is relative to the current node.
func scoreFromTT(score int16, ply int) int {
	s := int(score)
	if s >= winScore-1000 {
		return s - ply
	}
	if s <= -(winScore - 1000) {
		return s + ply
	}
	return s
}

// negamax is the recursive search core. Returns the score from the perspective
// of the side to move and the best move found, using alpha-beta pruning with
// negamax symmetry.
//
// TT integration: before move generation, probe the TT. If a usable entry is
// found (sufficient depth + matching bound), return immediately. Otherwise,
// use the stored move for ordering. After the search, store the result.
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

	// Repetition: threefold repetition is a draw. Cheap check — scans the
	// undo stack hashes bounded by halfmove clock.
	if p.IsRepetition() {
		return 0, types.Move{}
	}

	// TT probe: if we have a usable entry, return its score. Even if the
	// depth is insufficient, the stored move improves ordering.
	ply := p.Ply()
	var ttMove *types.Move
	if ctx.tt != nil {
		if entry, ok := ctx.tt.Probe(p.Hash); ok {
			if entry.Depth >= uint8(depth) {
				score := scoreFromTT(entry.Score, ply)
				switch entry.Flag {
				case ttExact:
					m, hasMove := engine.UnpackMove(entry.Move)
					if !hasMove {
						m = types.Move{}
					}
					return score, m
				case ttLower:
					if score >= beta {
						return score, types.Move{}
					}
				case ttUpper:
					if score <= alpha {
						return score, types.Move{}
					}
				}
			}
			if m, ok := engine.UnpackMove(entry.Move); ok {
				ttMove = &m
			}
		}
	}

	if depth == 0 {
		return Evaluate(p), types.Move{}
	}

	var ml engine.MoveList
	p.PseudoLegalMoves(&ml)
	orderMoves(&ml, ttMove)

	moverColor := sideColor(p)
	inCheck := p.IsInCheck(moverColor)
	best := negInf
	bestMove := types.Move{}
	initialAlpha := alpha

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

	// TT store: save the result with the appropriate bound flag.
	// Only store if not aborted (aborted results are unreliable).
	if ctx.tt != nil && !ctx.aborted {
		flag := ttExact
		switch {
		case best <= initialAlpha:
			flag = ttUpper
		case best >= beta:
			flag = ttLower
		}
		ctx.tt.Store(p.Hash, uint8(depth), scoreToTT(best, ply), engine.PackMove(bestMove), engine.TTFlag(flag))
	}

	return best, bestMove
}