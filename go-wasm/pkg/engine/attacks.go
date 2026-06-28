package engine

import "webassemble/pkg/types"

// kingColorIndex returns 0 for white, 1 for black — the index into
// Position.KingSquares.
func kingColorIndex(color types.Piece) int {
	if color == types.ColorBlack {
		return 1
	}
	return 0
}

// FindKing returns the board index of the king of the given color, or -1.
// Uses the cached KingSquares for O(1) lookup.
func (p *Position) FindKing(color types.Piece) int {
	return p.KingSquares[kingColorIndex(color)]
}

// IsInCheck returns true if the king of `color` is currently attacked.
func (p *Position) IsInCheck(color types.Piece) bool {
	kingIdx := p.FindKing(color)
	if kingIdx == -1 {
		return false
	}
	return p.attackersTo(kingIdx, oppositeColor(color)) != 0
}

// IsSquareAttacked reports whether `idx` is attacked by any piece of `byColor`.
// Delegates to attackersTo (bitboard-based) and checks if any attacker exists.
func (p *Position) IsSquareAttacked(idx int, byColor types.Piece) bool {
	return p.attackersTo(idx, byColor) != 0
}

// attackersTo returns a bitboard of all squares from which a piece of `byColor`
// attacks the square `idx`. This replaces the old reverse-scan approach with
// parallel bitboard operations: one AND per piece type, OR'd together.
//
// Pawn attacks: a white pawn on X attacks X+7 and X+9. So white pawns
// attacking `idx` are on idx-7 (if idx is not on file H) and idx-9 (if idx
// is not on file A). Black pawns attacking `idx` are on idx+9 and idx+7
// (mirrored). We shift a single-bit bitboard at `idx` in the reverse
// direction and AND with the enemy pawn bitboard.
//
// Knight/King: use precomputed attack tables — knightAttacks[idx] gives all
// squares a knight on idx attacks, so AND with enemy knights gives attackers.
//
// Sliders: use magic bitboard lookups — bishopAttacksBB(idx, occ) gives all
// squares a bishop on idx attacks, AND with enemy bishops+queens gives
// diagonal attackers. Same for rooks.
func (p *Position) attackersTo(idx int, byColor types.Piece) Bitboard {
	sqBB := Bitboard(1) << idx

	var pawns, knights, bishops, rooks, queens, king Bitboard
	if byColor == types.ColorWhite {
		pawns, knights, bishops, rooks, queens, king = p.WhitePawns, p.WhiteKnights, p.WhiteBishops, p.WhiteRooks, p.WhiteQueens, p.WhiteKing
	} else {
		pawns, knights, bishops, rooks, queens, king = p.BlackPawns, p.BlackKnights, p.BlackBishops, p.BlackRooks, p.BlackQueens, p.BlackKing
	}

	var attackers Bitboard

	// Pawn attackers: shift sqBB backward to find where attacking pawns would sit.
	// White pawns attack +7 (NW) and +9 (NE), so attackers of idx are at idx-7 and idx-9.
	// Black pawns attack -7 (SE) and -9 (SW), so attackers of idx are at idx+7 and idx+9.
	if byColor == types.ColorWhite {
		attackers |= (sqBB >> 7) & notA & pawns  // white pawn at sq-7 (NW), must not be on file A
		attackers |= (sqBB >> 9) & notH & pawns  // white pawn at sq-9 (NE), must not be on file H
	} else {
		attackers |= (sqBB << 9) & notA & pawns  // black pawn at sq+9 (SE), must not be on file A
		attackers |= (sqBB << 7) & notH & pawns  // black pawn at sq+7 (SW), must not be on file H
	}

	// Knight attackers.
	attackers |= knightAttacks[idx] & knights

	// King attackers.
	attackers |= kingAttacks[idx] & king

	// Bishop/queen attackers (diagonal sliders).
	diagAttackers := bishopAttacksBB(idx, p.Occupied)
	attackers |= diagAttackers & (bishops | queens)

	// Rook/queen attackers (orthogonal sliders).
	orthoAttackers := rookAttacksBB(idx, p.Occupied)
	attackers |= orthoAttackers & (rooks | queens)

	return attackers
}
