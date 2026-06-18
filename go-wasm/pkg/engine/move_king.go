package engine

import "webassemble/pkg/types"

var kingDirections = []int{-1, 1, 8, -8, 7, -7, 9, -9}

func GetMoveKing(piece types.Piece, i int, moves []types.Move) []types.Move {
	isWhite := piece&types.ColorWhite == types.ColorWhite
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

		if Board[target] == 0 {
			moves = append(moves, types.Move{From: i, To: target})
			continue
		}
		isEnemy := (isWhite && Board[target]&types.ColorBlack == types.ColorBlack) ||
			(!isWhite && Board[target]&types.ColorWhite == types.ColorWhite)
		if isEnemy {
			moves = append(moves, types.Move{From: i, To: target})
		}
	}
	rightsK, rightsQ, kingPos, rookK, rookQ := types.CastleWhiteK, types.CastleWhiteQ, 4, 7, 0
	if piece.Color() == types.ColorBlack {
		rightsK, rightsQ, kingPos, rookK, rookQ = types.CastleBlackK, types.CastleBlackQ, 60, 63, 56
	}

	if castlingRights&rightsK != 0 &&
		Board[kingPos].TypePiece() == types.King &&
		Board[kingPos].Color() == piece.Color() &&
		Board[rookK].TypePiece() == types.Rook &&
		Board[rookK].Color() == piece.Color() &&
		isEmpty([]int{kingPos + 1, kingPos + 2}) &&
		!IsSquareAttacked(kingPos, oppositeColor(piece.Color())) &&
		!IsSquareAttacked(kingPos+1, oppositeColor(piece.Color())) &&
		!IsSquareAttacked(kingPos+2, oppositeColor(piece.Color())) {
		moves = append(moves, types.Move{From: kingPos, To: kingPos + 2})
	}

	if castlingRights&rightsQ != 0 &&
		Board[kingPos].TypePiece() == types.King &&
		Board[kingPos].Color() == piece.Color() &&
		Board[rookQ].TypePiece() == types.Rook &&
		Board[rookQ].Color() == piece.Color() &&
		isEmpty([]int{kingPos - 1, kingPos - 2, kingPos - 3}) &&
		!IsSquareAttacked(kingPos, oppositeColor(piece.Color())) &&
		!IsSquareAttacked(kingPos-1, oppositeColor(piece.Color())) &&
		!IsSquareAttacked(kingPos-2, oppositeColor(piece.Color())) {
		moves = append(moves, types.Move{From: kingPos, To: kingPos - 2})
	}

	return moves
}

func isEmpty(squars []int) bool {
	for _, idx := range squars {
		if Board[idx] != 0 {
			return false
		}

	}
	return true
}
