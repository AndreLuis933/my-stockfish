package engine

import "webassemble/pkg/types"

// PseudoLegalCaptures generates only captures, en passant, and promotions
// (capturing or not — a pawn promoting to queen changes material balance).
// Used by quiescence search to extend past the depth horizon with only
// "noisy" moves, avoiding the horizon effect where a hanging piece appears
// safe just because the capture is past the depth cutoff.
//
// Writes into the caller-owned MoveList — zero heap allocation.
func (p *Position) PseudoLegalCaptures(ml *MoveList) {
	ml.Clear()
	for i, piece := range p.Board {
		if piece == 0 || piece.IsWhite() != p.WhiteToMove {
			continue
		}

		switch piece & types.TypeMask {
		case types.Pawn:
			p.capturePawn(piece, i, ml)
		case types.Rook:
			p.captureSlider(piece, i, ml, rookDirections[:], false)
		case types.Bishop:
			p.captureSlider(piece, i, ml, bishopDirections[:], true)
		case types.Queen:
			p.captureSlider(piece, i, ml, rookDirections[:], false)
			p.captureSlider(piece, i, ml, bishopDirections[:], true)
		case types.King:
			p.captureKing(piece, i, ml)
		case types.Knight:
			p.captureKnight(piece, i, ml)
		}
	}
}

// capturePawn generates only pawn captures, en passant, and capture-promotions.
func (p *Position) capturePawn(piece types.Piece, i int, ml *MoveList) {
	row := i / boardSize
	col := i % boardSize
	isWhite := piece&types.ColorWhite == types.ColorWhite
	myColor := piece.Color()

	dir, promotionRow := -boardSize, 1
	if isWhite {
		dir, promotionRow = boardSize, 6
	}

	canCaptureEnPassant := p.EnPassantCapture != -1 &&
		i/boardSize == p.EnPassantCapture/boardSize &&
		abs(i%boardSize-p.EnPassantCapture%boardSize) == 1

	for _, dc := range []int{1, -1} {
		if dc == 1 && col == boardSize-1 {
			continue
		}
		if dc == -1 && col == 0 {
			continue
		}
		t := i + dir + dc
		if !inBounds(t) {
			continue
		}
		target := p.Board[t]

		if canCaptureEnPassant && t == p.EnPassantTarget {
			ml.Add(types.Move{From: i, To: t, Flag: types.FlagEnPassant, Captured: p.Board[p.EnPassantCapture]})
			continue
		}
		if !piece.IsEnemy(target) {
			continue
		}
		if row == promotionRow {
			promotionPawn(i, t, myColor, target, ml)
			continue
		}
		ml.Add(types.Move{From: i, To: t, Flag: types.FlagNormal, Captured: target})
	}

	// Promotion pushes (non-capturing) — a pawn promoting to queen is a
	// material change worth searching in quiescence.
	if forward := i + dir; inBounds(forward) && p.Board[forward] == 0 && row == promotionRow {
		promotionPawn(i, forward, myColor, 0, ml)
	}
}

// captureSlider generates only captures along sliding directions.
func (p *Position) captureSlider(piece types.Piece, i int, ml *MoveList, directions []int, diagonal bool) {
	startRow := i / boardSize

	for _, dir := range directions {
		isHorizontal := dir == -1 || dir == 1

		for target := i + dir; inBounds(target); target += dir {
			if !diagonal && isHorizontal && target/boardSize != startRow {
				break
			}
			if diagonal {
				col := target % boardSize
				prevCol := (target - dir) % boardSize
				if abs(col-prevCol) != 1 {
					break
				}
			}

			if p.Board[target] == 0 {
				continue
			}

			if piece.IsEnemy(p.Board[target]) {
				ml.Add(types.Move{From: i, To: target, Flag: types.FlagNormal, Captured: p.Board[target]})
			}
			break
		}
	}
}

// captureKnight generates only knight captures.
func (p *Position) captureKnight(piece types.Piece, i int, ml *MoveList) {
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

		if piece.IsEnemy(p.Board[target]) {
			ml.Add(types.Move{From: i, To: target, Flag: types.FlagNormal, Captured: p.Board[target]})
		}
	}
}

// captureKing generates only king captures (no castling — castling is quiet).
func (p *Position) captureKing(piece types.Piece, i int, ml *MoveList) {
	startRow, startCol := i/boardSize, i%boardSize

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

		if piece.IsEnemy(p.Board[target]) {
			ml.Add(types.Move{From: i, To: target, Flag: types.FlagNormal, Captured: p.Board[target]})
		}
	}
}