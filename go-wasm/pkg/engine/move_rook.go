package engine

import "webassemble/pkg/types"

var rookDirections = []int{-boardSize, boardSize, -1, 1}

// MoveRook slides along ranks and files until it hits a piece or the edge.
func (p *Position) MoveRook(piece types.Piece, i int, ml *MoveList) {
	startRow := i / boardSize

	for _, dir := range rookDirections {
		isHorizontal := dir == -1 || dir == 1

		for target := i + dir; inBounds(target); target += dir {
			if isHorizontal && target/boardSize != startRow {
				break
			}

			if p.Board[target] == 0 {
				ml.Add(types.Move{From: i, To: target, Flag: types.FlagNormal})
				continue
			}

			if piece.IsEnemy(p.Board[target]) {
				ml.Add(types.Move{From: i, To: target, Flag: types.FlagNormal, Captured: p.Board[target]})
			}
			break
		}
	}
}