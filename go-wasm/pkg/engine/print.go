package engine

import (
	"fmt"
	"unicode"

	"webassemble/pkg/types"
)

func PrintBoard() {
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
			piece := Board[row*8+col]
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
