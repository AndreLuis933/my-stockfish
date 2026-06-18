package engine

import "webassemble/pkg/types"

var rookDirections = []int{-boardSize, boardSize, -1, 1}

// MoveRook slides along ranks and files until it hits a piece or the edge.
func (p *Position) MoveRook(piece types.Piece, i int, moves []types.Move) []types.Move {
	isWhite := piece&types.ColorWhite == types.ColorWhite
	startRow := i / boardSize

	for _, dir := range rookDirections {
		isHorizontal := dir == -1 || dir == 1

		for target := i + dir; inBounds(target); target += dir {
			if isHorizontal && target/boardSize != startRow {
				break
			}

			if p.Board[target] == 0 {
				moves = append(moves, types.Move{From: i, To: target})
				continue
			}

			isEnemy := (isWhite && p.Board[target]&types.ColorBlack == types.ColorBlack) ||
				(!isWhite && p.Board[target]&types.ColorWhite == types.ColorWhite)
			if isEnemy {
				moves = append(moves, types.Move{From: i, To: target})
			}
			break
		}
	}

	return moves
}