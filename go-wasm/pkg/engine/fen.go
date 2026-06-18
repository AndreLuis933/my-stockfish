package engine

import (
	"strings"
	"unicode"

	"webassemble/pkg/types"
)

// Standard starting position FEN.
//   rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1
const StartingFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

var pieceFromLetter = map[byte]types.Piece{
	'p': types.Pawn, 'n': types.Knight, 'b': types.Bishop,
	'r': types.Rook, 'q': types.Queen, 'k': types.King,
}

// LoadFen parses a FEN string into the global Game position.
// Kept as a free function for the WASM bridge and tests; delegates to the
// Position method.
//
// TODO: field 3 (en passant target square) is not yet parsed — MakeMove
// re-derives en passant state from the double push. Will be fixed in the
// FEN-cleanup step so mid-game positions load correctly.
func LoadFen(fen string) {
	Game.LoadFen(fen)
}

// LoadFen parses a FEN string into this position, resetting all state first.
func (p *Position) LoadFen(fen string) {
	p.reset()

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
				p.Board[index] = pieceFromLetter[byte(unicode.ToLower(rune(c)))] | types.ColorWhite
			case c >= 'a' && c <= 'z':
				p.Board[index] = pieceFromLetter[byte(unicode.ToLower(rune(c)))] | types.ColorBlack
			}
			row++
		}
	}
	p.WhiteToMove = fields[1] == "w"

	for _, right := range fields[2] {
		switch right {
		case 'K':
			p.CastlingRights |= types.CastleWhiteK
		case 'Q':
			p.CastlingRights |= types.CastleWhiteQ
		case 'k':
			p.CastlingRights |= types.CastleBlackK
		case 'q':
			p.CastlingRights |= types.CastleBlackQ
		}
	}
}