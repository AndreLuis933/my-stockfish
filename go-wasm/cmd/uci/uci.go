package main

import (
	"strconv"

	"webassemble/pkg/engine"
	"webassemble/pkg/types"
)

// squareToUCI converts a 0-63 board index to algebraic notation (e.g. "e4").
// The engine's board layout is rank-major with index 0 = a1, index 63 = h8:
// index = rank*8 + file, where rank 0 = rank 1, file 0 = file a.
func squareToUCI(idx int) string {
	file := idx % 8
	rank := idx / 8
	return string(rune('a'+file)) + strconv.Itoa(rank+1)
}

// uciToSquare converts algebraic notation (e.g. "e4") to a 0-63 index.
// Returns -1 on invalid input. Index = rank*8 + file (0 = a1).
func uciToSquare(s string) int {
	if len(s) != 2 {
		return -1
	}
	file := int(s[0] - 'a')
	rank := int(s[1] - '1') // 0-indexed: rank 1 → 0, rank 8 → 7
	if file < 0 || file > 7 || rank < 0 || rank > 7 {
		return -1
	}
	return rank*8 + file
}

// moveToUCI serializes an engine Move to UCI format: from+to plus an
// optional promotion letter (q/r/b/n). Example: "e2e4", "e7e8q".
func moveToUCI(m types.Move) string {
	s := squareToUCI(m.From) + squareToUCI(m.To)
	if m.Promotion != 0 {
		switch m.Promotion & types.TypeMask {
		case types.Queen:
			s += "q"
		case types.Rook:
			s += "r"
		case types.Bishop:
			s += "b"
		case types.Knight:
			s += "n"
		}
	}
	return s
}

// parseUCIMove parses a UCI move string (e.g. "e2e4", "e7e8q") against the
// legal moves of the current position and returns the matching Move.
// Returns ok=false if the string is not a legal move.
func (s *uciSession) parseUCIMove(uci string) (types.Move, bool) {
	if len(uci) < 4 || len(uci) > 5 {
		return types.Move{}, false
	}
	from := uciToSquare(uci[0:2])
	to := uciToSquare(uci[2:4])
	if from == -1 || to == -1 {
		return types.Move{}, false
	}

	var promoLetter byte
	if len(uci) == 5 {
		promoLetter = uci[4]
	}

	var legal engine.MoveList
	s.pos.LegalMoves(&legal)
	for i := 0; i < legal.Len(); i++ {
		m := legal.Get(i)
		if m.From != from || m.To != to {
			continue
		}
		if promoLetter == 0 {
			if m.Promotion == 0 {
				return m, true
			}
			continue
		}
		if m.Promotion == 0 {
			continue
		}
		var expected types.Piece
		switch promoLetter {
		case 'q':
			expected = types.Queen
		case 'r':
			expected = types.Rook
		case 'b':
			expected = types.Bishop
		case 'n':
			expected = types.Knight
		default:
			return types.Move{}, false
		}
		if m.Promotion&types.TypeMask == expected {
			return m, true
		}
	}
	return types.Move{}, false
}

// applyUCIMove parses and applies a UCI move string to the current position.
// Returns ok=false (and leaves the position unchanged) if the move is illegal.
func (s *uciSession) applyUCIMove(uci string) bool {
	m, ok := s.parseUCIMove(uci)
	if !ok {
		return false
	}
	s.pos.Make(m)
	return true
}