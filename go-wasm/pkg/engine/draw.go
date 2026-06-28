package engine

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
// Uses bitboard popcounts — no 64-square scan.
func (p *Position) IsInsufficientMaterial() bool {
	// Any pawn, rook, or queen → sufficient material.
	if p.WhitePawns != 0 || p.BlackPawns != 0 ||
		p.WhiteRooks != 0 || p.BlackRooks != 0 ||
		p.WhiteQueens != 0 || p.BlackQueens != 0 {
		return false
	}

	wN := popcount(p.WhiteKnights)
	bN := popcount(p.BlackKnights)
	wB := popcount(p.WhiteBishops)
	bB := popcount(p.BlackBishops)

	totalMinor := wN + bN + wB + bB

	// K vs K
	if totalMinor == 0 {
		return true
	}

	// K + minor vs K (one knight or one bishop total)
	if totalMinor == 1 {
		return true
	}

	// K + B vs K + B with same-color bishops
	if totalMinor == 2 && wB == 1 && bB == 1 {
		wbSq := bitscan(p.WhiteBishops)
		bbSq := bitscan(p.BlackBishops)
		wbColor := (wbSq/8 + wbSq%8) % 2
		bbColor := (bbSq/8 + bbSq%8) % 2
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