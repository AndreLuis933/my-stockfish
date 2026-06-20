package engine

import "webassemble/pkg/types"

var knightDirections = [8]int{-6, 6, 10, -10, 17, -17, 15, -15}

// MoveKnight generates the 8 L-shaped jumps, filtered by board edges.
func (p *Position) MoveKnight(piece types.Piece, i int, ml *MoveList) {
	startRow, startCol := i/boardSize, i%boardSize

	for _, dir := range knightDirections {
		target := i + dir
		if !inBounds(target) {
			continue
		}

		rowDiff := abs(target/boardSize - startRow)
		colDiff := abs(target%boardSize - startCol)

		if !((rowDiff == 1 && colDiff == 2) || (rowDiff == 2 && colDiff == 1)) {
			continue
		}

		if p.Board[target] == 0 {
			ml.Add(types.Move{From: i, To: target, Flag: types.FlagNormal})
			continue
		}

		if piece.IsEnemy(p.Board[target]) {
			ml.Add(types.Move{From: i, To: target, Flag: types.FlagNormal, Captured: p.Board[target]})
		}
	}
}