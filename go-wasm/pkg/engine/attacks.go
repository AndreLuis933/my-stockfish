package engine

import "webassemble/pkg/types"

func FindKing(color types.Piece) int {
	for i, p := range Board {
		if p&types.TypeMask == types.King && p.Color() == color {
			return i
		}
	}
	return -1
}

func IsSquareAttacked(idx int, byColor types.Piece) bool {
	row, col := idx/boardSize, idx%boardSize

	// 1. Pawn attacks — byColor pawns sit one rank "behind" relative to their direction
	pawnRowOffset := -1 // white pawns attack upward, so they sit at idx-7/-9
	if byColor == types.ColorBlack {
		pawnRowOffset = 1 // black pawns attack downward, sit at idx+7/+9
	}
	for dc := -1; dc <= 1; dc += 2 {
		r, c := row+pawnRowOffset, col+dc
		if r < 0 || r > 7 || c < 0 || c > 7 {
			continue
		}
		t := r*boardSize + c
		if Board[t].TypePiece() == types.Pawn && Board[t].Color() == byColor {
			return true
		}
	}

	// 2. Knight — reuse knightDirections + row/col diff validation (same as move_knight.go)
	for _, dir := range knightDirections {
		t := idx + dir
		if !inBounds(t) {
			continue
		}
		if abs(t/boardSize-row) == 2 && abs(t%boardSize-col) == 1 ||
			abs(t/boardSize-row) == 1 && abs(t%boardSize-col) == 2 {
			if Board[t].TypePiece() == types.Knight && Board[t].Color() == byColor {
				return true
			}
		}
	}

	// 3. King — 8 adjacent
	for _, dir := range kingDirections {
		t := idx + dir
		if !inBounds(t) {
			continue
		}
		if abs(t/boardSize-row) <= 1 && abs(t%boardSize-col) <= 1 {
			if Board[t].TypePiece() == types.King && Board[t].Color() == byColor {
				return true
			}
		}
	}

	for _, dir := range rookDirections {
		isHorizontal := dir == -1 || dir == 1

		for target := idx + dir; inBounds(target); target += dir {
			if isHorizontal && target/boardSize != row {
				break
			}
			if Board[target] == 0 {
				continue
			}

			if (Board[target].TypePiece() == types.Rook || Board[target].TypePiece() == types.Queen) && Board[target].Color() == byColor {
				return true
			}
			break
		}
	}

	for _, dir := range bishopDirections {
		prevCol := idx % boardSize

		for target := idx + dir; inBounds(target); target += dir {
			atuCol := target % boardSize

			if abs(atuCol-prevCol) != 1 {
				break
			}
			prevCol = atuCol

			if Board[target] == 0 {
				continue
			}

			if (Board[target].TypePiece() == types.Bishop || Board[target].TypePiece() == types.Queen) && Board[target].Color() == byColor {
				return true
			}
			break
		}
	}
	return false
}

func IsInCheck(color types.Piece) bool {
	kingIdx := FindKing(color)
	if kingIdx == -1 {
		return false
	}
	return IsSquareAttacked(kingIdx, oppositeColor(color))
}
