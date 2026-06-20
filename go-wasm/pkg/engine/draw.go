package engine

import "webassemble/pkg/types"

// IsRepetition returns true if the current position has occurred at least
// twice before in the current search tree (threefold repetition). Scans the
// undo stack hashes backwards in steps of 2 (only same-side-to-move
// positions can repeat), bounded by the halfmove clock — positions before
// the last irreversible move cannot repeat.
//
// This works for both the AI search (undoStack = search tree) and the
// frontend game (undoStack = game moves, since the frontend doesn't Unmake).
func (p *Position) IsRepetition() bool {
	current := p.Hash
	count := 1

	// Scan back through the undo stack, skipping the top entry (current
	// position). Step by 2 — only positions with the same side to move match.
	for i := p.undoPly - 2; i >= 0; i -= 2 {
		if p.undoStack[i].hash == current {
			count++
			if count >= 3 {
				return true
			}
		}
		// Beyond the halfmove clock, positions can't repeat (an irreversible
		// move happened).
		if p.undoPly-1-i >= p.HalfmoveClock {
			break
		}
	}
	return false
}

// IsTwoFoldRepetition returns true if the current position has occurred at
// least once before. Used by the AI to penalize (but not forbid) repetitions
// — a twofold is not yet a draw, but a third repetition would be.
func (p *Position) IsTwoFoldRepetition() bool {
	current := p.Hash
	for i := p.undoPly - 2; i >= 0; i -= 2 {
		if p.undoStack[i].hash == current {
			return true
		}
		if p.undoPly-1-i >= p.HalfmoveClock {
			break
		}
	}
	return false
}

// IsFiftyMoveRule returns true if 100 half-moves have passed without a pawn
// move or capture. The 50-move rule is an automatic draw.
func (p *Position) IsFiftyMoveRule() bool {
	return p.HalfmoveClock >= 100
}

// IsInsufficientMaterial returns true if neither side has enough material to
// force checkmate. The following combinations are draws:
//   - King vs King
//   - King + Bishop vs King
//   - King + Knight vs King
//   - King + Bishop vs King + Bishop (same color bishops)
//
// Zero-allocation: bishop square colors are tracked as int8 pairs, not slices.
func (p *Position) IsInsufficientMaterial() bool {
	var whiteBishopSq, blackBishopSq int
	whiteBishops := 0
	blackBishops := 0
	whiteKnights := 0
	blackKnights := 0
	whitePawns := 0
	blackPawns := 0
	whiteRooks := 0
	blackRooks := 0
	whiteQueens := 0
	blackQueens := 0

	for sq, piece := range p.Board {
		if piece == 0 {
			continue
		}
		isWhite := piece&types.ColorMask == types.ColorWhite
		switch piece & types.TypeMask {
		case types.Pawn:
			if isWhite {
				whitePawns++
			} else {
				blackPawns++
			}
		case types.Knight:
			if isWhite {
				whiteKnights++
			} else {
				blackKnights++
			}
		case types.Bishop:
			if isWhite {
				whiteBishops++
				whiteBishopSq = sq
			} else {
				blackBishops++
				blackBishopSq = sq
			}
		case types.Rook:
			if isWhite {
				whiteRooks++
			} else {
				blackRooks++
			}
		case types.Queen:
			if isWhite {
				whiteQueens++
			} else {
				blackQueens++
			}
		}
	}

	// Any pawn, rook, or queen → sufficient material.
	if whitePawns > 0 || blackPawns > 0 ||
		whiteRooks > 0 || blackRooks > 0 ||
		whiteQueens > 0 || blackQueens > 0 {
		return false
	}

	totalMinor := whiteKnights + blackKnights + whiteBishops + blackBishops

	// K vs K
	if totalMinor == 0 {
		return true
	}

	// K + minor vs K (one knight or one bishop total)
	if totalMinor == 1 {
		return true
	}

	// K + B vs K + B with same-color bishops
	if totalMinor == 2 && whiteBishops == 1 && blackBishops == 1 {
		wbColor := (whiteBishopSq/8 + whiteBishopSq%8) % 2
		bbColor := (blackBishopSq/8 + blackBishopSq%8) % 2
		return wbColor == bbColor
	}

	return false
}

// IsDraw returns true if the position is a draw by any rule: 50-move,
// threefold repetition, or insufficient material. This is the full draw check
// used by CurrentStatus and the AI search.
func (p *Position) IsDraw() bool {
	return p.IsFiftyMoveRule() || p.IsRepetition() || p.IsInsufficientMaterial()
}