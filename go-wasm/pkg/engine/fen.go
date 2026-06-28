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
			if p.Board[index]&types.TypeMask == types.King {
				if p.Board[index]&types.ColorMask == types.ColorWhite {
					p.KingSquares[0] = index
				} else {
					p.KingSquares[1] = index
				}
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

	// Build the bitboards from the mailbox (must run before eval init
	// so we can iterate piece bitboards instead of scanning 64 squares).
	p.updateBitboards()

	// Compute the initial incremental evaluation score by iterating
	// over piece bitboards (only set bits = actual pieces).
	p.EvalScore = 0
	p.evalFromBitboards()

	// Compute the initial Zobrist hash.
	p.Hash = p.ComputeHash()
}

// indexToSquare converts a board index (0-63) to algebraic notation (e.g. "e4").
// Inverse of squareToIndex.
func indexToSquare(index int) string {
	if index < 0 || index > 63 {
		return "-"
	}
	file := byte('a' + (index % 8))
	rank := byte('1' + (7 - index/8))
	return string([]byte{file, rank})
}

// castlingString builds the FEN castling-rights field from the bitmask.
// Order is always KQkq (white kingside, white queenside, black kingside,
// black queenside); "-" if no rights.
func (p *Position) castlingString() string {
	var sb strings.Builder
	if p.CastlingRights&types.CastleWhiteK != 0 {
		sb.WriteByte('K')
	}
	if p.CastlingRights&types.CastleWhiteQ != 0 {
		sb.WriteByte('Q')
	}
	if p.CastlingRights&types.CastleBlackK != 0 {
		sb.WriteByte('k')
	}
	if p.CastlingRights&types.CastleBlackQ != 0 {
		sb.WriteByte('q')
	}
	if sb.Len() == 0 {
		return "-"
	}
	return sb.String()
}

// FEN returns the standard 6-field FEN string for the current position.
// All fields are emitted: piece placement, side to move, castling rights,
// en passant target, halfmove clock, fullmove number.
//
// Board layout: index 0 = a8 (row 0 = rank 8 in our convention is WRONG —
// actually row 0 = rank 1 because LoadFen maps FEN rank 8 to targetRank=7).
// FEN requires rank 8 first, so we iterate rows from 7 down to 0.
func (p *Position) FEN() string {
	letters := map[types.Piece]byte{
		types.Pawn: 'p', types.Knight: 'n', types.Bishop: 'b',
		types.Rook: 'r', types.Queen: 'q', types.King: 'k',
	}
	var sb strings.Builder
	for row := 7; row >= 0; row-- {
		empty := 0
		for col := 0; col < 8; col++ {
			piece := p.Board[row*8+col]
			if piece == 0 {
				empty++
				continue
			}
			if empty > 0 {
				sb.WriteByte(byte('0' + empty))
				empty = 0
			}
			letter := letters[piece&types.TypeMask]
			if piece&types.ColorMask == types.ColorWhite {
				letter = byte(unicode.ToUpper(rune(letter)))
			}
			sb.WriteByte(letter)
		}
		if empty > 0 {
			sb.WriteByte(byte('0' + empty))
		}
		if row > 0 {
			sb.WriteByte('/')
		}
	}
	side := "w"
	if !p.WhiteToMove {
		side = "b"
	}
	ep := "-"
	if p.EnPassantTarget >= 0 {
		ep = indexToSquare(p.EnPassantTarget)
	}
	return sb.String() + " " + side + " " + p.castlingString() + " " + ep +
		" " + strconv.Itoa(p.HalfmoveClock) + " " + strconv.Itoa(p.FullmoveNumber)
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
