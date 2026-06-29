package engine

import "webassemble/pkg/types"

// MovePawn generates pawn moves: single push, double push, captures,
// en passant, and promotions (4 moves per promotable push: Q/N/B/R).
//
// Uses bitboard shifts for move computation. White pawns move toward higher
// indices (<< 8); black pawns move toward lower indices (>> 8).
func (p *Position) MovePawn(piece types.Piece, i int, ml *MoveList) {
	isWhite := piece&types.ColorWhite == types.ColorWhite
	myColor := piece.Color()
	row := i / boardSize

	pawnBB := Bitboard(1) << i

	var enemyPieces, empty Bitboard
	if isWhite {
		enemyPieces = p.BlackPieces
	} else {
		enemyPieces = p.WhitePieces
	}
	empty = p.Empty

	// Promotion rank: white promotes on row 7 (rank 8), black on row 0 (rank 1).
	promotionRow := 7
	if !isWhite {
		promotionRow = 0
	}

	if isWhite {
		// Single push.
		single := (pawnBB << 8) & empty
		if single != 0 {
			to := bitscan(single)
			toRow := to / boardSize
			if toRow == promotionRow {
				promotionPawn(i, to, myColor, 0, ml)
			} else {
				ml.Add(types.Move{From: uint8(i), To: uint8(to)})
			}
			// Double push from rank 2 (row 1).
			if row == 1 {
				double := (single << 8) & empty
				if double != 0 {
					ml.Add(types.Move{From: uint8(i), To: uint8(bitscan(double)), Flag: types.FlagDoublePush})
				}
			}
		}

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
	} else {
		// Single push.
		single := (pawnBB >> 8) & empty
		if single != 0 {
			to := bitscan(single)
			toRow := to / boardSize
			if toRow == promotionRow {
				promotionPawn(i, to, myColor, 0, ml)
			} else {
				ml.Add(types.Move{From: uint8(i), To: uint8(to)})
			}
			// Double push from rank 7 (row 6).
			if row == 6 {
				double := (single >> 8) & empty
				if double != 0 {
					ml.Add(types.Move{From: uint8(i), To: uint8(bitscan(double)), Flag: types.FlagDoublePush})
				}
			}
		}

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
	}
}

// promotionPawn appends the four promotion moves (Q, N, B, R) for a pawn
// reaching the last rank. `captured` is the piece on the destination square
// (0 for a push promotion, an enemy piece for a capture-promotion) — needed
// by Unmake to restore the board.
func promotionPawn(from, to int, color, captured types.Piece, ml *MoveList) {
	for _, promotion := range []types.Piece{types.Queen, types.Knight, types.Bishop, types.Rook} {
		ml.Add(types.Move{
			From:      uint8(from),
			To:        uint8(to),
			Promotion: promotion | color,
			Flag:      types.FlagPromotion,
			Captured:  captured,
		})
	}
}
