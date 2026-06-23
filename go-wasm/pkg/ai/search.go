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

// Search tuning constants.
const (
	// nullMoveReduction is the depth reduction applied during null-move
	// pruning. The classic value is 2 ("R=2"); deeper reduction prunes more
	// but risks missing zugzwang defenses.
	nullMoveReduction = 2

	// lmrFullMoves is the number of moves searched at full depth before LMR
	// kicks in. The PV move and captures are always full-depth.
	lmrFullMoves = 3

	// lmrMinDepth is the minimum remaining depth for LMR to apply. At very
	// shallow depths reducing further risks missing tactics.
	lmrMinDepth = 3

	// lmrReduction is the base depth reduction for late moves. The actual
	// reduction is this value plus a small bonus based on how late the move
	// is in the list.
	lmrReduction = 1
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

// hasNonPawnMaterial returns true if the side to move has any non-pawn,
// non-king material. Used to guard null-move pruning: in positions where
// the side to move has only pawns + king, zugzwang is more likely and the
// "pass" null move is unsafe.
func hasNonPawnMaterial(p *engine.Position) bool {
	moverColor := sideColor(p)
	for _, piece := range p.Board {
		if piece == 0 || piece.Color() != moverColor {
			continue
		}
		t := piece & types.TypeMask
		if t == types.Knight || t == types.Bishop || t == types.Rook || t == types.Queen {
			return true
		}
	}
	return false
}

// negamax is the recursive search core. Returns the score from the perspective
// of the side to move and the best move found, using alpha-beta pruning with
// negamax symmetry.
//
// Features, in order of application per node:
//   - TT probe: return early if a usable entry is found, else use TT move for ordering
//   - Check extension at the leaf (in check → search one more ply)
//   - Null-move pruning: pass once; if still ≥ beta, prune (skip in check / endgame)
//   - Move generation + ordering (TT move, captures by MVV, killers, history)
//   - Late move reductions: full depth for the first few moves, reduced for the rest
//   - Alpha-beta cutoffs with killer/history recording on quiet cutoffs
//   - TT store with bound classification
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

	ply := p.Ply()

	// Threefold repetition must be checked BEFORE the TT probe.
	// Without this, a position that is already a draw by repetition can
	// return a stale winning score from the TT (stored before the position
	// repeated), causing the engine to play into a 3-fold repetition while
	// thinking it's winning. The TT entry was correct when it was stored
	// (the position hadn't repeated yet), but it's wrong now.
	//
	// The 50-move rule is NOT checked here: a position can be both
	// "HalfmoveClock >= 100" and checkmate, and checkmate wins. The 50-move
	// check stays after the move loop where mate/stalemate is detected first.
	if p.IsRepetition() {
		return 0, types.Move{}
	}

	// TT probe: if we have a usable entry, return its score. Even if the
	// depth is insufficient, the stored move improves ordering.
	//
	// When the current position is a twofold repetition, the TT score may
	// be stale: the entry was stored when the position had only occurred
	// once, so children that are now 3-fold repetitions (score 0) were
	// scored as winning. Skip the TT probe in this case to force a fresh
	// search that correctly detects the 3-fold repetition in children.
	var ttMove *types.Move
	if ctx.tt != nil && !p.IsTwoFoldRepetition() {
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
	// at the depth horizon.
	moverColor := sideColor(p)
	inCheck := p.IsInCheck(moverColor)
	if depth == 0 {
		if inCheck {
			depth = 1
		} else {
			return quiescence(p, alpha, beta, ctx), types.Move{}
		}
	}

	// Null-move pruning: let the side to move "pass" (do nothing) and search
	// the opponent's reply at reduced depth. If even that is ≥ beta, the
	// position is so good we can prune — the side to move would do at least
	// as well with a real move.
	//
	// Conditions to avoid zugzwang false positives:
	//   - not in check (can't pass while in check)
	//   - side has non-pawn material (zugzwang unlikely)
	//   - skip at the root (ply 0) — we need a real best move
	//   - not a ply where we'd store a useless "null move" killer
	//   - depth > nullMoveReduction+1 so the reduced search is meaningful
	if !inCheck && ply > 0 && depth > nullMoveReduction+1 && hasNonPawnMaterial(p) && !ctx.disableNullMove {
		// Make a null move: flip the side, clear en passant, update hash.
		// We do this inline rather than via a Make() with a sentinel move
		// because Make expects a real move and would corrupt the board.
		oldEP := p.EnPassantTarget
		oldHash := p.Hash
		p.WhiteToMove = !p.WhiteToMove
		p.EnPassantTarget = -1
		p.Hash ^= engine.ZobristSideKey
		if oldEP >= 0 {
			p.Hash ^= engine.ZobristEPKeys[oldEP%8]
		}

		nullScoreValue, _ := negamax(p, depth-1-nullMoveReduction, -beta, -beta+1, ctx, nil)
		nullScore := -nullScoreValue

		// Unmake the null move.
		p.WhiteToMove = !p.WhiteToMove
		p.EnPassantTarget = oldEP
		p.Hash = oldHash

		if ctx.aborted {
			return 0, types.Move{}
		}
		if nullScore >= beta {
			return nullScore, types.Move{}
		}
	}

	var ml engine.MoveList
	p.PseudoLegalMoves(&ml)
	orderMoves(&ml, ttMove, &ctx.killers, &ctx.history, ply)

	best := negInf
	bestMove := types.Move{}
	initialAlpha := alpha

	for i := 0; i < ml.Len(); i++ {
		if ctx.aborted {
			break
		}
		m := ml.Get(i)
		isCapture := m.Captured != 0 || m.Flag == types.FlagEnPassant
		isPromotion := m.Flag == types.FlagPromotion
		p.Make(m)
		if p.IsInCheck(moverColor) {
			p.Unmake(m)
			continue
		}

		// Late move reductions (LMR): for moves after the first few, search
		// at reduced depth. If the reduced search improves alpha, re-search
		// at full depth to confirm. This trades a small risk of missing
		// tactics for a large node-count reduction.
		//
		// Conditions: depth deep enough, not a capture/promotion (those are
		// "noisy" and need full depth), not a killer (killers are good
		// candidates for full depth), and not the PV/TT move (i==0).
		reduced := false
		if i >= lmrFullMoves && depth >= lmrMinDepth && !isCapture && !isPromotion &&
			!ctx.killers.isKiller(ply, m) {
			// Search at depth-1-lmrReduction. Use a null-window around
			// alpha for the reduced search (since we only care if it
			// beats alpha, not the exact value).
			score, _ := negamax(p, depth-1-lmrReduction, -alpha-1, -alpha, ctx, nil)
			score = -score
			reduced = true
			if score <= alpha {
				// Reduced search didn't improve alpha — keep this score
				// as the result, no re-search needed.
				p.Unmake(m)
				if score > best {
					best = score
					bestMove = m
				}
				continue
			}
			// Reduced search beat alpha — re-search at full depth below.
		}

		score, _ := negamax(p, depth-1, -beta, -alpha, ctx, nil)
		score = -score
		_ = reduced
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
			// Beta cutoff — record the cutoff move in killers + history
			// if it's a quiet move. Captures are already well-ordered by
			// MVV and don't need heuristic help.
			if !isCapture {
				ctx.killers.storeKiller(ply, m)
				ctx.history.store(m, depth)
			}
			break
		}
	}

	if best == negInf {
		return noLegalMoveScore(inCheck, ply), types.Move{}
	}

	// 50-move rule: a claimable draw. If the side to move is losing (best < 0),
	// the draw is better — clamp to 0. If winning, play on. This is checked
	// after the move loop so checkmate/stalemate (no legal moves) is detected
	// first — a checkmated position is never a draw.
	if p.HalfmoveClock >= 100 && best < 0 {
		best = 0
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