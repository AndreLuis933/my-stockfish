package engine

import (
	"strings"
	"unicode"
	"webassemble/pkg/types"
)

// Fen
// rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1
var pieceFromLetter = map[byte]types.Piece{
	'p': types.Pawn, 'n': types.Knight, 'b': types.Bishop,
	'r': types.Rook, 'q': types.Queen, 'k': types.King,
}

func LoadFen(fen string) {
	fields := strings.Fields(fen)
	ranks := strings.Split(fields[0], "/")
	for i, rank := range ranks {
		targetRank := 7 - i
		row := 0
		for _, c := range rank {
			index := targetRank*8 + row
			switch {
			case c >= '1' && c <= '8':
				row += int(c - '0')
				continue
			case c >= 'A' && c <= 'Z':
				Board[index] = pieceFromLetter[byte(unicode.ToLower(rune(c)))] | types.ColorWhite
			case c >= 'a' && c <= 'z':
				Board[index] = pieceFromLetter[byte(unicode.ToLower(rune(c)))] | types.ColorBlack
			}
			row++
		}
	}
	whiteToMove = fields[1] == "w"

	for _, right := range fields[2] {
		switch right {
		case 'K':
			castlingRights |= types.CastleWhiteK
		case 'Q':
			castlingRights |= types.CastleWhiteQ
		case 'k':
			castlingRights |= types.CastleBlackK
		case 'q':
			castlingRights |= types.CastleBlackQ
		}
	}

}
