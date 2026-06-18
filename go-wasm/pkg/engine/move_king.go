package engine

import "webassemble/pkg/types"

var kingDirections = []int{-1, 1, 8, -8, 7, -7, 9, -9}

func GetMoveKing(piece types.Piece, i int, moves []types.Move) []types.Move {
	isWhite := piece&types.ColorWhite == types.ColorWhite

	for _, dir := range kingDirections {
		target := i + dir
		if !inBounds(target) {
			continue
		}

		if Board[target] == 0 {
			moves = append(moves, types.Move{From: i, To: target})
			continue
		}
		isEnemy := (isWhite && Board[target]&types.ColorBlack == types.ColorBlack) ||
			(!isWhite && Board[target]&types.ColorWhite == types.ColorWhite)
		if isEnemy {
			moves = append(moves, types.Move{From: i, To: target})
		}

	}

	return moves
}