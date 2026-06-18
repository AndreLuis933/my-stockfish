package engine

import (
	"webassemble/pkg/types"
)

var Board types.Board

const boardSize = 8

var enPassantCapture = -1
var enPassantTarget = -1

func GetMovePawn(piece types.Piece, i int, moves []types.Move) []types.Move {
	row := i / boardSize
	col := i % boardSize
	isWhite := piece&types.ColorWhite == types.ColorWhite

	dir, startRow, enemyColor := -boardSize, 6, types.ColorWhite
	if isWhite {
		dir, startRow, enemyColor = boardSize, 1, types.ColorBlack
	}

	if forward := i + dir; inBounds(forward) && Board[forward] == 0 {
		moves = append(moves, types.Move{From: i, To: forward})

		if row == startRow {
			if double := i + 2*dir; inBounds(double) && Board[double] == 0 {
				moves = append(moves, types.Move{From: i, To: double})
			}
		}
	}

	canCaptureEnPassant := enPassantCapture != -1 &&
		i/boardSize == enPassantCapture/boardSize &&
		abs(i%boardSize-enPassantCapture%boardSize) == 1

	if col != boardSize-1 {
		if t := i + dir + 1; inBounds(t) && (Board[t]&enemyColor == enemyColor || (canCaptureEnPassant && t == enPassantTarget)) {
			moves = append(moves, types.Move{From: i, To: t})
		}
	}

	if col != 0 {
		if t := i + dir - 1; inBounds(t) && (Board[t]&enemyColor == enemyColor || (canCaptureEnPassant && t == enPassantTarget)) {
			moves = append(moves, types.Move{From: i, To: t})
		}
	}

	return moves
}

var rookDirections = []int{-boardSize, boardSize, -1, 1}

func GetMoveRook(piece types.Piece, i int, moves []types.Move) []types.Move {
	isWhite := piece&types.ColorWhite == types.ColorWhite
	startRow := i / boardSize

	for _, dir := range rookDirections {
		isHorizontal := dir == -1 || dir == 1

		for target := i + dir; inBounds(target); target += dir {
			// impede "vazar" pra linha de cima/baixo nas direções horizontais
			if isHorizontal && target/boardSize != startRow {
				break
			}

			if Board[target] == 0 {
				moves = append(moves, types.Move{From: i, To: target})
				continue // casa vazia, segue deslizando
			}

			isEnemy := (isWhite && Board[target]&types.ColorBlack == types.ColorBlack) ||
				(!isWhite && Board[target]&types.ColorWhite == types.ColorWhite)
			if isEnemy {
				moves = append(moves, types.Move{From: i, To: target})
			}
			break // achou peça (aliada ou inimiga), para de deslizar nessa direção
		}
	}

	return moves
}

var bishopDirections = []int{-boardSize - 1, -boardSize + 1, boardSize - 1, boardSize + 1}

func GetMoveBishop(piece types.Piece, i int, moves []types.Move) []types.Move {
	isWhite := piece&types.ColorWhite == types.ColorWhite

	for _, dir := range bishopDirections {
		prevCol := i % boardSize

		for target := i + dir; inBounds(target); target += dir {
			col := target % boardSize

			// se a coluna não andou exatamente 1 casa, vazou de borda
			if abs(col-prevCol) != 1 {
				break
			}
			prevCol = col

			if Board[target] == 0 {
				moves = append(moves, types.Move{From: i, To: target})
				continue
			}

			isEnemy := (isWhite && Board[target]&types.ColorBlack == types.ColorBlack) ||
				(!isWhite && Board[target]&types.ColorWhite == types.ColorWhite)
			if isEnemy {
				moves = append(moves, types.Move{From: i, To: target})
			}
			break
		}
	}

	return moves
}

var kingDirections = []int{-1, 1, 8, -8, 7, -7, 9, -9}

func GetMoveKing(piece types.Piece, i int, moves []types.Move) []types.Move {
	isWhite := piece&types.ColorWhite == types.ColorWhite

	for _, dir := range kingDirections {
		target := i + dir
		if !inBounds(target) {
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

	return moves
}

var knightDirections = []int{-6, 6, 10, -10, 17, -17, 15, -15}

func GetMoveKnight(piece types.Piece, i int, moves []types.Move) []types.Move {
	isWhite := piece&types.ColorWhite == types.ColorWhite
	startRow, startCol := i/boardSize, i%boardSize

	for _, dir := range knightDirections {
		target := i + dir
		if !inBounds(target) {
			continue
		}

		rowDiff := abs(target/boardSize - startRow)
		colDiff := abs(target%boardSize - startCol)

		// movimento de cavalo só é válido se formar um "L": (1,2) ou (2,1)
		if !((rowDiff == 1 && colDiff == 2) || (rowDiff == 2 && colDiff == 1)) {
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

	return moves
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func inBounds(idx int) bool {
	return idx >= 0 && idx < boardSize*boardSize
}

func GetValidMoves() []types.Move {
	var moves []types.Move
	for i, piece := range Board {

		if piece&types.Pawn == types.Pawn {
			moves = GetMovePawn(piece, i, moves)
		}
		if piece&types.Rook == types.Rook {
			moves = GetMoveRook(piece, i, moves)
		}
		if piece&types.Bishop == types.Bishop {
			moves = GetMoveBishop(piece, i, moves)
		}
		if piece&types.Queen == types.Queen {
			moves = GetMoveRook(piece, i, moves)
			moves = GetMoveBishop(piece, i, moves)
		}
		if piece&types.King == types.King {
			moves = GetMoveKing(piece, i, moves)
		}
		if piece&types.Knight == types.Knight {
			moves = GetMoveKnight(piece, i, moves)
		}

	}
	return moves
}

func MakeMovement(from, to int) {
	var piece = Board[from]
	if piece&types.Pawn == types.Pawn && to == enPassantTarget && enPassantCapture != -1 {
		Board[enPassantCapture] = 0
	}
	Board[from] = 0
	Board[to] = piece

	enPassantCapture, enPassantTarget = -1, -1

	if piece&types.Pawn == types.Pawn {
		diff := to - from
		if diff == 2*boardSize || diff == -2*boardSize {
			enPassantCapture = to
			enPassantTarget = (from + to) / 2
		}
	}
}
