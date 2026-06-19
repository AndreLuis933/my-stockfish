package engine

import "webassemble/pkg/types"

var knightDirections = []int{-6, 6, 10, -10, 17, -17, 15, -15}

// MoveKnight generates the 8 L-shaped jumps, filtered by board edges.
func (p *Position) MoveKnight(piece types.Piece, i int, moves []types.Move) []types.Move {
	isWhite := piece&types.ColorWhite == types.ColorWhite
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
			moves = append(moves, types.Move{From: i, To: target, Flag: types.FlagNormal})
			continue
		}

		isEnemy := (isWhite && p.Board[target]&types.ColorBlack == types.ColorBlack) ||
			(!isWhite && p.Board[target]&types.ColorWhite == types.ColorWhite)
		if isEnemy {
			moves = append(moves, types.Move{From: i, To: target, Flag: types.FlagNormal, Captured: p.Board[target]})
		}
	}

	return moves
}