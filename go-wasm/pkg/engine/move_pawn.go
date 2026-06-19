package engine

import "webassemble/pkg/types"

// MovePawn generates pawn moves: single push, double push, captures,
// en passant, and promotions (4 moves per promotable push: Q/N/B/R).
//
// Direction convention: white pawns move "up" the board (toward index 0,
// because rank 8 is index 0). dir = -boardSize means -8.
func (p *Position) MovePawn(piece types.Piece, i int, ml *MoveList) {
	row := i / boardSize
	col := i % boardSize
	isWhite := piece&types.ColorWhite == types.ColorWhite
	myColor := piece.Color()

	dir, startRow, promotionRow := -boardSize, 6, 1
	if isWhite {
		dir, startRow, promotionRow = boardSize, 1, 6
	}

	// Single + Double forward push (only if the square is empty).
	if forward := i + dir; inBounds(forward) && p.Board[forward] == 0 {
		if row == promotionRow {
			promotionPawn(i, forward, myColor, 0, ml)
		} else {
			ml.Add(types.Move{From: i, To: forward})
		}

		if row == startRow {
			if double := i + 2*dir; p.Board[double] == 0 {
				ml.Add(types.Move{From: i, To: double, Flag: types.FlagDoublePush})
			}
		}
	}

	// En passant availability: our pawn must sit next to the pawn that just
	// did a double push (same rank, adjacent file).
	canCaptureEnPassant := p.EnPassantCapture != -1 &&
		i/boardSize == p.EnPassantCapture/boardSize &&
		abs(i%boardSize-p.EnPassantCapture%boardSize) == 1

	// Diagonal captures (both directions). En passant and capture-promotions
	for _, dc := range []int{1, -1} {
		if dc == 1 && col == boardSize-1 {
			continue
		}
		if dc == -1 && col == 0 {
			continue
		}
		t := i + dir + dc
		if !inBounds(t) {
			continue
		}
		target := p.Board[t]

		if canCaptureEnPassant && t == p.EnPassantTarget { // en passant (target square is empty)
			ml.Add(types.Move{From: i, To: t, Flag: types.FlagEnPassant, Captured: p.Board[p.EnPassantCapture]})
			continue
		}
		if !piece.IsEnemy(target) { // rejects empty squares and friendly pieces
			continue
		}
		if row == promotionRow { // capture-promotion
			promotionPawn(i, t, myColor, target, ml)
			continue
		}
		ml.Add(types.Move{From: i, To: t, Flag: types.FlagNormal, Captured: target}) // normal capture
	}
}

// promotionPawn appends the four promotion moves (Q, N, B, R) for a pawn
// reaching the last rank. `captured` is the piece on the destination square
// (0 for a push promotion, an enemy piece for a capture-promotion) — needed
// by Unmake to restore the board.
func promotionPawn(from, to int, color, captured types.Piece, ml *MoveList) {
	for _, promotion := range []types.Piece{types.Queen, types.Knight, types.Bishop, types.Rook} {
		ml.Add(types.Move{
			From:      from,
			To:        to,
			Promotion: promotion | color,
			Flag:      types.FlagPromotion,
			Captured:  captured,
		})
	}
}