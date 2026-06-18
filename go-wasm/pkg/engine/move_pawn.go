package engine

import "webassemble/pkg/types"

func GetMovePawn(piece types.Piece, i int, moves []types.Move) []types.Move {
	row := i / boardSize
	col := i % boardSize
	isWhite := piece&types.ColorWhite == types.ColorWhite
	enemyColor := oppositeColor(piece)
	myColor := PieceColor(piece)

	dir, startRow, promotionRow := -boardSize, 6, 1
	if isWhite {
		dir, startRow, promotionRow = boardSize, 1, 6
	}

	if forward := i + dir; inBounds(forward) && Board[forward] == 0 {
		if row == promotionRow {
			moves = promotionPawn(i, forward, myColor, moves)

		} else {

			moves = append(moves, types.Move{From: i, To: forward})
		}

		if row == startRow {
			if double := i + 2*dir; inBounds(double) && Board[double] == 0 {
				moves = append(moves, types.Move{From: i, To: double})
			}
		}
	}

	canCaptureEnPassant := enPassantCapture != -1 &&
		i/boardSize == enPassantCapture/boardSize &&
		abs(i%boardSize-enPassantCapture%boardSize) == 1

	if col != boardSize-1 {
		if t := i + dir + 1; inBounds(t) && (Board[t]&enemyColor == enemyColor || (canCaptureEnPassant && t == enPassantTarget)) {
			if row == promotionRow {
				moves = promotionPawn(i, t, myColor, moves)

			} else {

				moves = append(moves, types.Move{From: i, To: t})
			}
		}
	}

	if col != 0 {
		if t := i + dir - 1; inBounds(t) && (Board[t]&enemyColor == enemyColor || (canCaptureEnPassant && t == enPassantTarget)) {
			if row == promotionRow {
				moves = promotionPawn(i, t, myColor, moves)

			} else {

				moves = append(moves, types.Move{From: i, To: t})
			}
		}
	}

	return moves
}

func promotionPawn(from, to int, color types.Piece, moves []types.Move) []types.Move {
	for _, promoton := range []types.Piece{types.Queen, types.Knight, types.Bishop, types.Rook} {
		moves = append(moves, types.Move{From: from, To: to, Promotion: PiecePtr(promoton | color)})
	}

	return moves
}
