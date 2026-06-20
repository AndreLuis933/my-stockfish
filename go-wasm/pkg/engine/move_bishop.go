package engine

import "webassemble/pkg/types"

var bishopDirections = [4]int{-boardSize - 1, -boardSize + 1, boardSize - 1, boardSize + 1}

// MoveBishop slides along the 4 diagonals until it hits a piece or the edge.
func (p *Position) MoveBishop(piece types.Piece, i int, ml *MoveList) {
	for _, dir := range bishopDirections {
		prevCol := i % boardSize

		for target := i + dir; inBounds(target); target += dir {
			col := target % boardSize

			if abs(col-prevCol) != 1 {
				break
			}
			prevCol = col

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