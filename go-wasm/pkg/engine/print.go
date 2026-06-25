package engine

import (
	"fmt"
	"strings"
	"unicode"

	"webassemble/pkg/types"
)

// PrintBoard prints the global Game position as an 8x8 ASCII grid (debug only).
func PrintBoard() {
	Game.PrintBoard()
}

// PrintBoard prints this position as an 8x8 ASCII grid (debug only).
// Uppercase = white, lowercase = black, '.' = empty.
func (p *Position) PrintBoard() {
	letters := map[types.Piece]byte{
		types.Pawn:   'p',
		types.Knight: 'n',
		types.Bishop: 'b',
		types.Rook:   'r',
		types.Queen:  'q',
		types.King:   'k',
	}

	for row := 0; row < 8; row++ {
		for col := 0; col < 8; col++ {
			piece := p.Board[row*8+col]
			letter := letters[piece&types.TypeMask]

			if letter == 0 {
				fmt.Print(". ")
				continue
			}
			if piece&types.ColorMask == types.ColorWhite {
				letter = byte(unicode.ToUpper(rune(letter)))
			}
			fmt.Printf("%c ", letter)
		}
		fmt.Println()
	}
}

// FenSnapshot returns a compact FEN string for the current position, for
// logging/debug purposes only. Not optimized for hot paths.
func (p *Position) FenSnapshot() string {
	letters := map[types.Piece]byte{
		types.Pawn: 'p', types.Knight: 'n', types.Bishop: 'b',
		types.Rook: 'r', types.Queen: 'q', types.King: 'k',
	}
	var sb strings.Builder
	for row := 0; row < 8; row++ {
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
		if row < 7 {
			sb.WriteByte('/')
		}
	}
	side := "w"
	if !p.WhiteToMove {
		side = "b"
	}
	return fmt.Sprintf("%s %s - - %d %d", sb.String(), side, p.HalfmoveClock, p.FullmoveNumber)
}