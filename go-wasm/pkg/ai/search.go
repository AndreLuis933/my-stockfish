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

	// 50-move / repetition draws are checked after move generation below:
	// checkmate/stalemate must be detected first, since a position can be
	// both "HalfmoveClock >= 100" and checkmate, and checkmate wins.

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

	// Check extension at the leaf: when the side to move is in check at
	// depth 0, search one more ply with full move generation instead of
	// dropping into quiescence. Quiescence only searches captures and
	// returns stand-pat eval — it cannot detect checkmate (no legal moves
	// while in check). Without this extension, the engine misses mate-in-1
	// at the depth horizon: Qb2# would score as a normal eval, not +winScore.
	moverColor := sideColor(p)
	inCheck := p.IsInCheck(moverColor)
	if depth == 0 {
		if inCheck {
			depth = 1
		} else {
			return quiescence(p, alpha, beta, ctx), types.Move{}
		}
	}

	var ml engine.MoveList
	p.PseudoLegalMoves(&ml)
	orderMoves(&ml, ttMove)

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
		return noLegalMoveScore(inCheck, ply), types.Move{}
	}

	// 50-move rule and threefold repetition are claimable draws: the side
	// to move can claim the draw (score 0), but only if it's better than
	// their best move. If checkmate is available (best > 0), the side plays
	// on and wins. If all moves are worse than a draw, the side claims it.
	// This is checked after the move loop so mate/stalemate (no legal moves)
	// is detected first — noLegalMoveScore handles those.
	if p.HalfmoveClock >= 100 || p.IsRepetition() {
		if best < 0 {
			best = 0
		}
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