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
func (p *Position) MoveKing(piece types.Piece, i int, moves []types.Move) []types.Move {
	isWhite := piece&types.ColorWhite == types.ColorWhite
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
			moves = append(moves, types.Move{From: i, To: target, Flag: types.FlagNormal})
			continue
		}
		isEnemy := (isWhite && p.Board[target]&types.ColorBlack == types.ColorBlack) ||
			(!isWhite && p.Board[target]&types.ColorWhite == types.ColorWhite)
		if isEnemy {
			moves = append(moves, types.Move{From: i, To: target, Flag: types.FlagNormal, Captured: p.Board[target]})
		}
	}

	// Castling — white and black use different squares, so we pick the right
	// set of constants once based on the king color.
	rightsK, rightsQ, kingPos, rookK, rookQ := types.CastleWhiteK, types.CastleWhiteQ, 4, 7, 0
	if piece.Color() == types.ColorBlack {
		rightsK, rightsQ, kingPos, rookK, rookQ = types.CastleBlackK, types.CastleBlackQ, 60, 63, 56
	}

	enemy := oppositeColor(piece.Color())

	if p.CastlingRights&rightsK != 0 &&
		p.Board[kingPos].TypePiece() == types.King &&
		p.Board[kingPos].Color() == piece.Color() &&
		p.Board[rookK].TypePiece() == types.Rook &&
		p.Board[rookK].Color() == piece.Color() &&
		p.isEmpty([]int{kingPos + 1, kingPos + 2}) &&
		!p.IsSquareAttacked(kingPos, enemy) &&
		!p.IsSquareAttacked(kingPos+1, enemy) &&
		!p.IsSquareAttacked(kingPos+2, enemy) {
		moves = append(moves, types.Move{From: kingPos, To: kingPos + 2, Flag: types.FlagCastleK})
	}

	if p.CastlingRights&rightsQ != 0 &&
		p.Board[kingPos].TypePiece() == types.King &&
		p.Board[kingPos].Color() == piece.Color() &&
		p.Board[rookQ].TypePiece() == types.Rook &&
		p.Board[rookQ].Color() == piece.Color() &&
		p.isEmpty([]int{kingPos - 1, kingPos - 2, kingPos - 3}) &&
		!p.IsSquareAttacked(kingPos, enemy) &&
		!p.IsSquareAttacked(kingPos-1, enemy) &&
		!p.IsSquareAttacked(kingPos-2, enemy) {
		moves = append(moves, types.Move{From: kingPos, To: kingPos - 2, Flag: types.FlagCastleQ})
	}

	return moves
}

func (p *Position) isEmpty(squares []int) bool {
	for _, idx := range squares {
		if p.Board[idx] != 0 {
			return false
		}
	}
	return true
}