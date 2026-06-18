package engine

import "webassemble/pkg/types"

// FindKing returns the board index of the king of the given color, or -1.
func (p *Position) FindKing(color types.Piece) int {
	for i, piece := range p.Board {
		if piece&types.TypeMask == types.King && piece.Color() == color {
			return i
		}
	}
	return -1
}

// IsSquareAttacked reports whether `idx` is attacked by any piece of `byColor`.
// It scans outward from the target square (reverse scan): pawn diagonals,
// knight L-jumps, king one-steps, then rook/bishop sliding rays.
//
// Reverse scan (from the target, looking outward) is faster than iterating
// all enemy pieces because we stop as soon as we find one attacker per ray.
func (p *Position) IsSquareAttacked(idx int, byColor types.Piece) bool {
	row, col := idx/boardSize, idx%boardSize

	// 1. Pawn attacks — a pawn of `byColor` sits one rank "behind" its attack
	//    direction. White pawns attack upward, so they sit at idx-9 / idx-7.
	pawnRowOffset := -1
	if byColor == types.ColorBlack {
		pawnRowOffset = 1
	}
	for dc := -1; dc <= 1; dc += 2 {
		r, c := row+pawnRowOffset, col+dc
		if r < 0 || r > 7 || c < 0 || c > 7 {
			continue
		}
		t := r*boardSize + c
		if p.Board[t].TypePiece() == types.Pawn && p.Board[t].Color() == byColor {
			return true
		}
	}

	// 2. Knight L-jumps.
	for _, dir := range knightDirections {
		t := idx + dir
		if !inBounds(t) {
			continue
		}
		if abs(t/boardSize-row) == 2 && abs(t%boardSize-col) == 1 ||
			abs(t/boardSize-row) == 1 && abs(t%boardSize-col) == 2 {
			if p.Board[t].TypePiece() == types.Knight && p.Board[t].Color() == byColor {
				return true
			}
		}
	}

	// 3. King one-step (8 adjacent squares).
	for _, dir := range kingDirections {
		t := idx + dir
		if !inBounds(t) {
			continue
		}
		if abs(t/boardSize-row) <= 1 && abs(t%boardSize-col) <= 1 {
			if p.Board[t].TypePiece() == types.King && p.Board[t].Color() == byColor {
				return true
			}
		}
	}

	// 4. Rook / Queen sliding (orthogonal rays).
	for _, dir := range rookDirections {
		isHorizontal := dir == -1 || dir == 1

		for target := idx + dir; inBounds(target); target += dir {
			if isHorizontal && target/boardSize != row {
				break
			}
			if p.Board[target] == 0 {
				continue
			}

			if (p.Board[target].TypePiece() == types.Rook || p.Board[target].TypePiece() == types.Queen) && p.Board[target].Color() == byColor {
				return true
			}
			break
		}
	}

	// 5. Bishop / Queen sliding (diagonal rays).
	for _, dir := range bishopDirections {
		prevCol := idx % boardSize

		for target := idx + dir; inBounds(target); target += dir {
			curCol := target % boardSize

			if abs(curCol-prevCol) != 1 {
				break
			}
			prevCol = curCol

			if p.Board[target] == 0 {
				continue
			}

			if (p.Board[target].TypePiece() == types.Bishop || p.Board[target].TypePiece() == types.Queen) && p.Board[target].Color() == byColor {
				return true
			}
			break
		}
	}
	return false
}

// IsInCheck returns true if the king of `color` is currently attacked.
func (p *Position) IsInCheck(color types.Piece) bool {
	kingIdx := p.FindKing(color)
	if kingIdx == -1 {
		return false
	}
	return p.IsSquareAttacked(kingIdx, oppositeColor(color))
}