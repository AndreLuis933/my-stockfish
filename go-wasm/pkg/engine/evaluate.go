package engine

import "webassemble/pkg/types"

const (
	PawnValue   = 100
	KnightValue = 320
	BishopValue = 330
	RookValue   = 500
	QueenValue  = 900
	KingValue   = 20000
)

func MaterialValue(piece types.Piece) int {
	switch piece & types.TypeMask {
	case types.Pawn:
		return PawnValue
	case types.Knight:
		return KnightValue
	case types.Bishop:
		return BishopValue
	case types.Rook:
		return RookValue
	case types.Queen:
		return QueenValue
	case types.King:
		return KingValue
	}
	return 0
}

var PawnTable = [64]int{
	0, 0, 0, 0, 0, 0, 0, 0,
	50, 50, 50, 50, 50, 50, 50, 50,
	10, 10, 20, 30, 30, 20, 10, 10,
	5, 5, 10, 25, 25, 10, 5, 5,
	0, 0, 0, 20, 20, 0, 0, 0,
	5, -5, -10, 0, 0, -10, -5, 5,
	5, 10, 10, -20, -20, 10, 10, 5,
	0, 0, 0, 0, 0, 0, 0, 0,
}

var KnightTable = [64]int{
	-50, -40, -30, -30, -30, -30, -40, -50,
	-40, -20, 0, 0, 0, 0, -20, -40,
	-30, 0, 10, 15, 15, 10, 0, -30,
	-30, 5, 15, 20, 20, 15, 5, -30,
	-30, 0, 15, 20, 20, 15, 0, -30,
	-30, 5, 10, 15, 15, 10, 5, -30,
	-40, -20, 0, 5, 5, 0, -20, -40,
	-50, -40, -30, -30, -30, -30, -40, -50,
}

var BishopTable = [64]int{
	-20, -10, -10, -10, -10, -10, -10, -20,
	-10, 0, 0, 0, 0, 0, 0, -10,
	-10, 0, 5, 10, 10, 5, 0, -10,
	-10, 5, 5, 10, 10, 5, 5, -10,
	-10, 0, 10, 10, 10, 10, 0, -10,
	-10, 10, 10, 10, 10, 10, 10, -10,
	-10, 5, 0, 0, 0, 0, 5, -10,
	-20, -10, -10, -10, -10, -10, -10, -20,
}

var RookTable = [64]int{
	0, 0, 0, 0, 0, 0, 0, 0,
	5, 10, 10, 10, 10, 10, 10, 5,
	-5, 0, 0, 0, 0, 0, 0, -5,
	-5, 0, 0, 0, 0, 0, 0, -5,
	-5, 0, 0, 0, 0, 0, 0, -5,
	-5, 0, 0, 0, 0, 0, 0, -5,
	-5, 0, 0, 0, 0, 0, 0, -5,
	0, 0, 0, 5, 5, 0, 0, 0,
}

var QueenTable = [64]int{
	-20, -10, -10, -5, -5, -10, -10, -20,
	-10, 0, 0, 0, 0, 0, 0, -10,
	-10, 0, 5, 5, 5, 5, 0, -10,
	-5, 0, 5, 5, 5, 5, 0, -5,
	0, 0, 5, 5, 5, 5, 0, -5,
	-10, 5, 5, 5, 5, 5, 0, -10,
	-10, 0, 5, 0, 0, 0, 0, -10,
	-20, -10, -10, -5, -5, -10, -10, -20,
}

var KingTable = [64]int{
	-30, -40, -40, -50, -50, -40, -40, -30,
	-30, -40, -40, -50, -50, -40, -40, -30,
	-30, -40, -40, -50, -50, -40, -40, -30,
	-30, -40, -40, -50, -50, -40, -40, -30,
	-20, -30, -30, -40, -40, -30, -30, -20,
	-10, -20, -20, -20, -20, -20, -20, -10,
	20, 20, 0, 0, 0, 0, 20, 20,
	20, 30, 10, 0, 0, 10, 30, 20,
}

var pstTables = [6][64]int{
	PawnTable, KnightTable, BishopTable, RookTable, QueenTable, KingTable,
}

func pstTableIndex(pieceType types.Piece) int {
	switch pieceType {
	case types.Pawn:
		return 0
	case types.Knight:
		return 1
	case types.Bishop:
		return 2
	case types.Rook:
		return 3
	case types.Queen:
		return 4
	case types.King:
		return 5
	}
	return -1
}

func pstValue(piece types.Piece, square int) int {
	isWhite := piece&types.ColorMask == types.ColorWhite
	idx := square
	if !isWhite {
		idx = square ^ 56
	}
	tableIdx := pstTableIndex(piece & types.TypeMask)
	if tableIdx < 0 {
		return 0
	}
	return pstTables[tableIdx][idx]
}

func pieceTotalValue(piece types.Piece, square int) int {
	return MaterialValue(piece) + pstValue(piece, square)
}

func signedPieceValue(piece types.Piece, square int) int {
	if piece&types.ColorMask == types.ColorWhite {
		return pieceTotalValue(piece, square)
	}
	return -pieceTotalValue(piece, square)
}