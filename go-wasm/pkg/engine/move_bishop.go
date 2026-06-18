package engine

import "webassemble/pkg/types"

var bishopDirections = []int{-boardSize - 1, -boardSize + 1, boardSize - 1, boardSize + 1}

func GetMoveBishop(piece types.Piece, i int, moves []types.Move) []types.Move {
	isWhite := piece&types.ColorWhite == types.ColorWhite

	for _, dir := range bishopDirections {
		prevCol := i % boardSize

		for target := i + dir; inBounds(target); target += dir {
			col := target % boardSize

			if abs(col-prevCol) != 1 {
				break
			}
			prevCol = col

			if Board[target] == 0 {
				moves = append(moves, types.Move{From: i, To: target})
				continue
			}

			isEnemy := (isWhite && Board[target]&types.ColorBlack == types.ColorBlack) ||
				(!isWhite && Board[target]&types.ColorWhite == types.ColorWhite)
			if isEnemy {
				moves = append(moves, types.Move{From: i, To: target})
			}
			break
		}
	}

	return moves
}