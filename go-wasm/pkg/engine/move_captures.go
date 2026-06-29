package engine

import "webassemble/pkg/types"

// PseudoLegalCaptures generates only captures, en passant, and promotions
// (capturing or not — a pawn promoting to queen changes material balance).
// Used by quiescence search to extend past the depth horizon with only
// "noisy" moves, avoiding the horizon effect where a hanging piece appears
// safe just because the capture is past the depth cutoff.
//
// Uses bitboard operations: magic lookups for sliders, precomputed tables
// for knights/kings, shift-based for pawns. Only iterates over actual pieces.
func (p *Position) PseudoLegalCaptures(ml *MoveList) {
	ml.Clear()

	var pawns, knights, bishops, rooks, queens, king, ownPieces, enemyPieces Bitboard
	var color types.Piece
	if p.WhiteToMove {
		pawns, knights, bishops, rooks, queens, king = p.WhitePawns, p.WhiteKnights, p.WhiteBishops, p.WhiteRooks, p.WhiteQueens, p.WhiteKing
		ownPieces, enemyPieces = p.WhitePieces, p.BlackPieces
		color = types.ColorWhite
	} else {
		pawns, knights, bishops, rooks, queens, king = p.BlackPawns, p.BlackKnights, p.BlackBishops, p.BlackRooks, p.BlackQueens, p.BlackKing
		ownPieces, enemyPieces = p.BlackPieces, p.WhitePieces
		color = types.ColorBlack
	}

	// Pawn captures + en passant + promotion pushes.
	bb := pawns
	for bb != 0 {
		i := bitscan(bb)
		bb &= bb - 1
		p.capturePawnBB(color|types.Pawn, i, ml, enemyPieces)
	}

	// Knight captures.
	bb = knights
	for bb != 0 {
		i := bitscan(bb)
		bb &= bb - 1
		targets := knightAttacks[i] & enemyPieces
		for targets != 0 {
			to := bitscan(targets)
			targets &= targets - 1
			ml.Add(types.Move{From: uint8(i), To: uint8(to), Flag: types.FlagNormal, Captured: p.Board[to]})
		}
	}

	// Bishop captures.
	bb = bishops
	for bb != 0 {
		i := bitscan(bb)
		bb &= bb - 1
		targets := bishopAttacksBB(i, p.Occupied) & enemyPieces
		for targets != 0 {
			to := bitscan(targets)
			targets &= targets - 1
			ml.Add(types.Move{From: uint8(i), To: uint8(to), Flag: types.FlagNormal, Captured: p.Board[to]})
		}
	}

	// Rook captures.
	bb = rooks
	for bb != 0 {
		i := bitscan(bb)
		bb &= bb - 1
		targets := rookAttacksBB(i, p.Occupied) & enemyPieces
		for targets != 0 {
			to := bitscan(targets)
			targets &= targets - 1
			ml.Add(types.Move{From: uint8(i), To: uint8(to), Flag: types.FlagNormal, Captured: p.Board[to]})
		}
	}

	// Queen captures (rook + bishop attacks).
	bb = queens
	for bb != 0 {
		i := bitscan(bb)
		bb &= bb - 1
		targets := (rookAttacksBB(i, p.Occupied) | bishopAttacksBB(i, p.Occupied)) & enemyPieces
		for targets != 0 {
			to := bitscan(targets)
			targets &= targets - 1
			ml.Add(types.Move{From: uint8(i), To: uint8(to), Flag: types.FlagNormal, Captured: p.Board[to]})
		}
	}

	// King captures.
	if king != 0 {
		i := bitscan(king)
		targets := kingAttacks[i] & enemyPieces
		for targets != 0 {
			to := bitscan(targets)
			targets &= targets - 1
			ml.Add(types.Move{From: uint8(i), To: uint8(to), Flag: types.FlagNormal, Captured: p.Board[to]})
		}
	}

	_ = ownPieces
}

// capturePawnBB generates pawn captures, en passant, and promotion pushes
// using bitboard shifts.
func (p *Position) capturePawnBB(piece types.Piece, i int, ml *MoveList, enemyPieces Bitboard) {
	isWhite := piece&types.ColorWhite == types.ColorWhite
	myColor := piece.Color()
	row := i / boardSize
	pawnBB := Bitboard(1) << i

	promotionRow := 7
	if !isWhite {
		promotionRow = 0
	}

	if isWhite {
		// Captures: NW (<< 7, file-1) and NE (<< 9, file+1).
		captures := ((pawnBB & notA) << 7) & enemyPieces
		captures |= ((pawnBB & notH) << 9) & enemyPieces
		for captures != 0 {
			to := bitscan(captures)
			captures &= captures - 1
			toRow := to / boardSize
			if toRow == promotionRow {
				promotionPawn(i, to, myColor, p.Board[to], ml)
			} else {
				ml.Add(types.Move{From: uint8(i), To: uint8(to), Flag: types.FlagNormal, Captured: p.Board[to]})
			}
		}

		// En passant.
		if p.EnPassantTarget != -1 {
			epBB := ((pawnBB & notA) << 7) | ((pawnBB & notH) << 9)
			if epBB&(1<<p.EnPassantTarget) != 0 {
				ml.Add(types.Move{From: uint8(i), To: uint8(p.EnPassantTarget), Flag: types.FlagEnPassant, Captured: p.Board[p.EnPassantCapture]})
			}
		}

		// Promotion pushes (non-capturing) — material change worth searching.
		if row == promotionRow-1 { // rank 7 (row 6) for white
			push := (pawnBB << 8) & p.Empty
			if push != 0 {
				promotionPawn(i, bitscan(push), myColor, 0, ml)
			}
		}
	} else {
		// Captures: SE (>> 7, file+1) and SW (>> 9, file-1).
		captures := ((pawnBB & notH) >> 7) & enemyPieces
		captures |= ((pawnBB & notA) >> 9) & enemyPieces
		for captures != 0 {
			to := bitscan(captures)
			captures &= captures - 1
			toRow := to / boardSize
			if toRow == promotionRow {
				promotionPawn(i, to, myColor, p.Board[to], ml)
			} else {
				ml.Add(types.Move{From: uint8(i), To: uint8(to), Flag: types.FlagNormal, Captured: p.Board[to]})
			}
		}

		// En passant.
		if p.EnPassantTarget != -1 {
			epBB := ((pawnBB & notH) >> 7) | ((pawnBB & notA) >> 9)
			if epBB&(1<<p.EnPassantTarget) != 0 {
				ml.Add(types.Move{From: uint8(i), To: uint8(p.EnPassantTarget), Flag: types.FlagEnPassant, Captured: p.Board[p.EnPassantCapture]})
			}
		}

		// Promotion pushes (non-capturing).
		if row == promotionRow+1 { // rank 2 (row 1) for black
			push := (pawnBB >> 8) & p.Empty
			if push != 0 {
				promotionPawn(i, bitscan(push), myColor, 0, ml)
			}
		}
	}
}
