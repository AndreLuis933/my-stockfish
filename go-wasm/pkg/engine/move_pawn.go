package engine

import "webassemble/pkg/types"

// MovePawn generates pawn moves: single push, double push, captures,
// en passant, and promotions (4 moves per promotable push: Q/N/B/R).
//
// Direction convention: white pawns move "up" the board (toward index 0,
// because rank 8 is index 0). dir = -boardSize means -8.
func (p *Position) MovePawn(piece types.Piece, i int, moves []types.Move) []types.Move {
	row := i / boardSize
	col := i % boardSize
	isWhite := piece&types.ColorWhite == types.ColorWhite
	enemyColor := oppositeColor(piece)
	myColor := pieceColor(piece)

	dir, startRow, promotionRow := -boardSize, 6, 1
	if isWhite {
		dir, startRow, promotionRow = boardSize, 1, 6
	}

	// Single + double forward push (only if the square is empty).
	if forward := i + dir; inBounds(forward) && p.Board[forward] == 0 {
		if row == promotionRow {
			moves = promotionPawn(i, forward, myColor, moves)
		} else {
			moves = append(moves, types.Move{From: i, To: forward})
		}

		if row == startRow {
			if double := i + 2*dir; inBounds(double) && p.Board[double] == 0 {
				moves = append(moves, types.Move{From: i, To: double})
			}
		}
	}

	// En passant availability: our pawn must sit next to the pawn that just
	// did a double push (same rank, adjacent file).
	canCaptureEnPassant := p.EnPassantCapture != -1 &&
		i/boardSize == p.EnPassantCapture/boardSize &&
		abs(i%boardSize-p.EnPassantCapture%boardSize) == 1

	// Capture to the right (toward higher file).
	if col != boardSize-1 {
		if t := i + dir + 1; inBounds(t) && (p.Board[t]&enemyColor == enemyColor || (canCaptureEnPassant && t == p.EnPassantTarget)) {
			if row == promotionRow {
				moves = promotionPawn(i, t, myColor, moves)
			} else {
				moves = append(moves, types.Move{From: i, To: t})
			}
		}
	}

	// Capture to the left (toward lower file).
	if col != 0 {
		if t := i + dir - 1; inBounds(t) && (p.Board[t]&enemyColor == enemyColor || (canCaptureEnPassant && t == p.EnPassantTarget)) {
			if row == promotionRow {
				moves = promotionPawn(i, t, myColor, moves)
			} else {
				moves = append(moves, types.Move{From: i, To: t})
			}
		}
	}

	return moves
}

// promotionPawn appends the four promotion moves (Q, N, B, R) for a pawn
// reaching the last rank. It is a free function (no Position state needed).
func promotionPawn(from, to int, color types.Piece, moves []types.Move) []types.Move {
	for _, promotion := range []types.Piece{types.Queen, types.Knight, types.Bishop, types.Rook} {
		moves = append(moves, types.Move{From: from, To: to, Promotion: PiecePtr(promotion | color)})
	}
	return moves
}