package engine

import (
	"strconv"
	"strings"
	"unicode"

	"webassemble/pkg/types"
)

// Standard starting position FEN.
//
//	rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1
//
// FEN fields:
//  0. Piece placement (rank 8 → rank 1, file a → h)
//  1. Side to move ("w" or "b")
//  2. Castling rights (KQkq subset, or "-")
//  3. En passant target square (e.g. "d6"), or "-"
//  4. Halfmove clock (plies since last pawn move or capture — 50-move rule)
//  5. Fullmove number (starts at 1, increments after black's move)
const StartingFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

var pieceFromLetter = map[byte]types.Piece{
	'p': types.Pawn, 'n': types.Knight, 'b': types.Bishop,
	'r': types.Rook, 'q': types.Queen, 'k': types.King,
}

// LoadFen parses a FEN string into the global Game position.
// Kept as a free function for the WASM bridge and tests; delegates to the
// Position method.
func LoadFen(fen string) {
	Game.LoadFen(fen)
}

// LoadFen parses a FEN string into this position, resetting all state first.
// All 6 FEN fields are parsed; missing trailing fields default to sane values
// (halfmove=0, fullmove=1) so partial FENs still work.
func (p *Position) LoadFen(fen string) {
	p.reset()

	fields := strings.Fields(fen)

	// Field 0 — piece placement.
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

	// Field 1 — side to move.
	p.WhiteToMove = fields[1] == "w"

	// Field 2 — castling rights.
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

	// Field 3 — en passant target square (e.g. "d6") or "-".
	//
	// FEN gives the square *behind* the pawn that just double-pushed (the
	// landing square of an en passant capture). We store that as
	// EnPassantTarget. We also derive EnPassantCapture — the actual pawn's
	// square — which sits one rank "back" from the target toward the color
	// that moved. If the target is on rank 6 (row 2, e.g. d6 = index 19), a
	// white pawn moved d2-d4, so the pawn is on rank 5 (row 3) = target + 8.
	// If the target is on rank 3 (row 5, e.g. d3 = index 43), a black pawn
	// moved d7-d5, so the pawn is on rank 4 (row 4) = target - 8.
	if len(fields) > 3 && fields[3] != "-" {
		if idx := squareToIndex(fields[3]); idx != -1 {
			p.EnPassantTarget = idx
			targetRow := idx / boardSize
			switch targetRow {
			case 2: // rank 6 → white pushed, pawn one row below (target + 8)
				p.EnPassantCapture = idx + boardSize
			case 5: // rank 3 → black pushed, pawn one row above (target - 8)
				p.EnPassantCapture = idx - boardSize
			}
		}
	}

	// Field 4 — halfmove clock (plies since last pawn move or capture).
	if len(fields) > 4 {
		if clock, err := strconv.Atoi(fields[4]); err == nil {
			p.HalfmoveClock = clock
		}
	}

	// Field 5 — fullmove number (starts at 1, increments after black's move).
	if len(fields) > 5 {
		if num, err := strconv.Atoi(fields[5]); err == nil {
			p.FullmoveNumber = num
		}
	} else {
		p.FullmoveNumber = 1
	}
}

// squareToIndex converts algebraic notation (e.g. "e4", "d6") to a board
// index (0-63), or returns -1 if the string is invalid.
//
// Algebraic: file is a-h (column 0-7), rank is 1-8. Our board index 0 = a8
// (rank 8, row 0), index 56 = a1 (rank 1, row 7). After '1' subtraction,
// rank is 0-indexed (0-7), so row = 7 - rank. Example: e4 = file 'e' (col 4)
// + rank 4 (0-indexed 3) → row 7-3=4 → index 4*8 + 4 = 36.
func squareToIndex(square string) int {
	if len(square) != 2 {
		return -1
	}
	file := int(square[0] - 'a')
	rank := int(square[1] - '1') // 0-indexed: rank 1 → 0, rank 8 → 7
	if file < 0 || file > 7 || rank < 0 || rank > 7 {
		return -1
	}
	row := 7 - rank // rank 1 (index 0) → row 7, rank 8 (index 7) → row 0
	return row*boardSize + file
}
