package engine

import "webassemble/pkg/types"

var kingDirections = []int{-1, 1, 8, -8, 7, -7, 9, -9}

// MoveKing generates one-step king moves plus castling (kingside & queenside).
//
// Castling checks all 6 FIDE conditions:
//  1. Castling rights still present (tracked in p.CastlingRights)
//  2. King and rook on their original squares
//  3. Squares between king and rook are empty
//  4. King is not currently in check
//  5. King does not pass through an attacked square
//  6. King does not land on an attacked square
//
// The rook move itself is applied in MakeMove (not here).
func (p *Position) MoveKing(piece types.Piece, i int, ml *MoveList) {
	startRow, startCol := i/boardSize, i%boardSize

	// Normal one-step king moves.
	for _, dir := range kingDirections {
		target := i + dir
		if !inBounds(target) {
			continue
		}

		rowDiff := abs(target/boardSize - startRow)
		colDiff := abs(target%boardSize - startCol)
		if rowDiff > 1 || colDiff > 1 {
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

	p.generateCastling(piece, ml)
}