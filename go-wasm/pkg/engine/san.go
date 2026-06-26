package engine

import (
	"fmt"
	"strings"

	"webassemble/pkg/types"
)

// sanSquare converts a board index (0-63) to algebraic notation.
// The engine stores rank 1 at index 0-7 (opposite of standard FEN order),
// so rank = index/8 + 1, file = index%8.
func sanSquare(index int) string {
	if index < 0 || index > 63 {
		return "-"
	}
	file := byte('a' + (index % 8))
	rank := byte('1' + (index / 8))
	return string([]byte{file, rank})
}

// sanToIndex converts algebraic notation to a board index, matching the
// engine's layout (index 0 = a1, index 63 = h8).
func sanToIndex(square string) int {
	if len(square) != 2 {
		return -1
	}
	file := int(square[0] - 'a')
	rank := int(square[1] - '1')
	if file < 0 || file > 7 || rank < 0 || rank > 7 {
		return -1
	}
	return rank*8 + file
}

var pieceLetters = map[types.Piece]string{
	types.King:   "K",
	types.Queen:  "Q",
	types.Rook:   "R",
	types.Bishop: "B",
	types.Knight: "N",
}

var promotionLetters = map[types.Piece]string{
	types.Queen:  "Q",
	types.Rook:   "R",
	types.Bishop: "B",
	types.Knight: "N",
}

// ToSan converts a legal move to Standard Algebraic Notation.
// The position must be the state BEFORE the move is made.
// The move must be a legal move in the current position.
func (p *Position) ToSan(m types.Move) (string, error) {
	piece := p.Board[m.From]
	if piece == 0 {
		return "", fmt.Errorf("san: no piece at square %d", m.From)
	}

	pieceType := piece & types.TypeMask

	// Castling
	if pieceType == types.King && abs(m.To-m.From) == 2 {
		base := "O-O"
		if m.To < m.From {
			base = "O-O-O"
		}
		return p.appendCheckSuffix(base, m), nil
	}

	isCapture := p.Board[m.To] != 0 || m.Flag == types.FlagEnPassant

	var sb strings.Builder

	if pieceType == types.Pawn {
		if isCapture {
			sb.WriteByte(byte('a' + (m.From % 8)))
			sb.WriteByte('x')
		}
		sb.WriteString(sanSquare(m.To))
		if m.Promotion != 0 {
			promoType := m.Promotion & types.TypeMask
			if letter, ok := promotionLetters[promoType]; ok {
				sb.WriteByte('=')
				sb.WriteString(letter)
			}
		}
	} else {
		letter, ok := pieceLetters[pieceType]
		if !ok {
			return "", fmt.Errorf("san: unknown piece type %d", pieceType)
		}
		sb.WriteString(letter)

		disamb := p.disambiguation(m, pieceType)
		sb.WriteString(disamb)

		if isCapture {
			sb.WriteByte('x')
		}
		sb.WriteString(sanSquare(m.To))
	}

	return p.appendCheckSuffix(sb.String(), m), nil
}

// disambiguation computes the minimal disambiguation string for a non-pawn,
// non-king move: file, rank, or full square if needed to distinguish from
// other pieces of the same type that can also move to the target square.
func (p *Position) disambiguation(m types.Move, pieceType types.Piece) string {
	piece := p.Board[m.From]
	color := piece.Color()
	var candidates []int

	var ml MoveList
	p.PseudoLegalMoves(&ml)

	for i := 0; i < ml.n; i++ {
		mv := ml.moves[i]
		if mv.From == m.From || mv.To != m.To {
			continue
		}
		cp := p.Board[mv.From]
		if cp&types.TypeMask != pieceType || cp.Color() != color {
			continue
		}
		p.Make(mv)
		inCheck := p.IsInCheck(color)
		p.Unmake(mv)
		if !inCheck {
			candidates = append(candidates, mv.From)
		}
	}
	if len(candidates) == 0 {
		return ""
	}

	fromFile := m.From % 8
	fromRank := m.From / 8

	sameFile := false
	sameRank := false
	for _, c := range candidates {
		if c%8 == fromFile {
			sameFile = true
		}
		if c/8 == fromRank {
			sameRank = true
		}
	}

	if !sameFile {
		return string(byte('a' + fromFile))
	}
	if !sameRank {
		return string(byte('1' + fromRank))
	}
	return sanSquare(m.From)
}

// appendCheckSuffix makes the move on the position, checks if it gives
// check or checkmate, appends "+" or "#", then unmakes.
func (p *Position) appendCheckSuffix(san string, m types.Move) string {
	p.Make(m)
	defer p.Unmake(m)

	// After Make, the side to move is the opponent. Check if the opponent
	// (the side that received the move) is in check.
	if p.IsInCheck(p.colorOfSide()) {
		var ml MoveList
		p.LegalMoves(&ml)
		if ml.n == 0 {
			return san + "#"
		}
		return san + "+"
	}
	return san
}

// SanToMove matches a SAN string to a legal move in the current position.
// Returns the matching Move or an error if no legal move matches.
func (p *Position) SanToMove(san string) (types.Move, error) {
	clean := stripCheckSuffix(san)

	var ml MoveList
	p.LegalMoves(&ml)

	for i := 0; i < ml.n; i++ {
		m := ml.moves[i]
		generated, err := p.ToSan(m)
		if err != nil {
			continue
		}
		if stripCheckSuffix(generated) == clean {
			return m, nil
		}
	}

	return types.Move{}, fmt.Errorf("san: no legal move matches %q", san)
}

func stripCheckSuffix(san string) string {
	s := strings.TrimRight(san, "+#!?")
	return s
}