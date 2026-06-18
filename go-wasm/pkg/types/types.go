package types

// Piece uint8:
// bits 0-5: type (one-hot)
// bits 6-7: color (00=none, 01=white, 10=black)
type Piece uint8

const (
	Pawn   Piece = 1 << 0
	Knight Piece = 1 << 1
	Bishop Piece = 1 << 2
	Rook   Piece = 1 << 3
	Queen  Piece = 1 << 4
	King   Piece = 1 << 5
)

const (
	ColorNone  Piece = 0b00 << 6
	ColorWhite Piece = 0b01 << 6
	ColorBlack Piece = 0b10 << 6
)

const (
	TypeMask  Piece = 0b00111111
	ColorMask Piece = 0b11000000
)

const Sliders Piece = Bishop | Rook | Queen

type Board [64]Piece

type Move struct {
	From int `json:"from"`
	To   int `json:"to"`
	Promotion *Piece `json:"promotion,omitempty"`
}
