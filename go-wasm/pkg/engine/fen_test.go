package engine

import (
	"testing"

	"webassemble/pkg/types"
)

func TestLoadFenEnPassantTarget(t *testing.T) {
	// White just played d2-d4. FEN field 3 = "d6" (the e.p. target square).
	// The pawn that double-pushed is on d5 (one rank below the target).
	// d6 = file d (3), rank 6 → (8-6)*8+3 = 19
	// d5 = file d (3), rank 5 → (8-5)*8+3 = 27
	loadFEN(t, "rnbqkbnr/pppppppp/8/8/3P4/8/PPP1PPPP/RNBQKBNR b KQkq d6 0 1")

	if Game.EnPassantTarget != 19 {
		t.Errorf("expected EnPassantTarget = 19 (d6), got %d", Game.EnPassantTarget)
	}
	if Game.EnPassantCapture != 27 {
		t.Errorf("expected EnPassantCapture = 27 (d5), got %d", Game.EnPassantCapture)
	}
}

func TestLoadFenEnPassantTargetBlack(t *testing.T) {
	// Black just played d7-d5. FEN field 3 = "d3" (the e.p. target square).
	// The pawn that double-pushed is on d4 (one rank above the target).
	// d3 = file d (3), rank 3 → (7-2)*8+3 = 43
	// d4 = file d (3), rank 4 → (7-3)*8+3 = 35
	loadFEN(t, "rnbqkbnr/ppp1pppp/8/3p4/8/8/PPPPPPPP/RNBQKBNR w KQkq d3 0 2")

	if Game.EnPassantTarget != 43 {
		t.Errorf("expected EnPassantTarget = 43 (d3), got %d", Game.EnPassantTarget)
	}
	if Game.EnPassantCapture != 35 {
		t.Errorf("expected EnPassantCapture = 35 (d4), got %d", Game.EnPassantCapture)
	}
}

func TestLoadFenEnPassantNone(t *testing.T) {
	loadFEN(t, "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
	if Game.EnPassantTarget != -1 {
		t.Errorf("expected EnPassantTarget = -1, got %d", Game.EnPassantTarget)
	}
	if Game.EnPassantCapture != -1 {
		t.Errorf("expected EnPassantCapture = -1, got %d", Game.EnPassantCapture)
	}
}

func TestLoadFenHalfmoveClock(t *testing.T) {
	loadFEN(t, "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 42 7")
	if Game.HalfmoveClock != 42 {
		t.Errorf("expected HalfmoveClock = 42, got %d", Game.HalfmoveClock)
	}
}

func TestLoadFenFullmoveNumber(t *testing.T) {
	loadFEN(t, "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 7")
	if Game.FullmoveNumber != 7 {
		t.Errorf("expected FullmoveNumber = 7, got %d", Game.FullmoveNumber)
	}
}

func TestLoadFenMissingTrailingFields(t *testing.T) {
	// Partial FEN without halfmove/fullmove — should default to 0 and 1.
	loadFEN(t, "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq -")
	if Game.HalfmoveClock != 0 {
		t.Errorf("expected default HalfmoveClock = 0, got %d", Game.HalfmoveClock)
	}
	if Game.FullmoveNumber != 1 {
		t.Errorf("expected default FullmoveNumber = 1, got %d", Game.FullmoveNumber)
	}
}

func TestSquareToIndex(t *testing.T) {
	cases := []struct {
		square string
		want   int
	}{
		{"a1", 56},
		{"h1", 63},
		{"a8", 0},
		{"h8", 7},
		{"e4", 36},
		{"d6", 19},
		{"d3", 43},
		{"i1", -1},   // invalid file
		{"a9", -1},   // invalid rank
		{"", -1},     // too short
		{"abc", -1},  // too long
	}
	for _, c := range cases {
		t.Run(c.square, func(t *testing.T) {
			if got := squareToIndex(c.square); got != c.want {
				t.Errorf("squareToIndex(%q) = %d, want %d", c.square, got, c.want)
			}
		})
	}
}

// TestMakeUnmakeHalfmoveClock verifies the clock resets on pawn moves and
// captures, and increments otherwise — and that Unmake restores it.
func TestMakeUnmakeHalfmoveClock(t *testing.T) {
	// Start: clock = 0.
	loadFEN(t, "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")

	// 1. e2-e4 (pawn move) → clock resets to 0.
	move := types.Move{From: 12, To: 28, Flag: types.FlagDoublePush}
	Game.Make(move)
	if Game.HalfmoveClock != 0 {
		t.Errorf("after pawn move, expected clock 0, got %d", Game.HalfmoveClock)
	}
	Game.Unmake(move)
	if Game.HalfmoveClock != 0 {
		t.Errorf("after unmake, expected clock 0, got %d", Game.HalfmoveClock)
	}

	// Set up a position with a non-pawn, non-capture move available.
	loadFEN(t, "4k3/8/8/8/8/8/8/4K2R w K - 5 10") // white rook h1, clock 5

	// Rook h1-h2 (quiet move) → clock increments to 6.
	rookMove := types.Move{From: 7, To: 15, Flag: types.FlagNormal}
	Game.Make(rookMove)
	if Game.HalfmoveClock != 6 {
		t.Errorf("after quiet rook move, expected clock 6, got %d", Game.HalfmoveClock)
	}
	Game.Unmake(rookMove)
	if Game.HalfmoveClock != 5 {
		t.Errorf("after unmake of rook move, expected clock 5, got %d", Game.HalfmoveClock)
	}
}

func TestMakeUnmakeFullmoveNumber(t *testing.T) {
	loadFEN(t, "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")

	if Game.FullmoveNumber != 1 {
		t.Fatalf("expected starting fullmove = 1, got %d", Game.FullmoveNumber)
	}

	// White moves e2-e4 → fullmove stays 1 (increments only after black).
	whiteMove := types.Move{From: 12, To: 28, Flag: types.FlagDoublePush}
	Game.Make(whiteMove)
	if Game.FullmoveNumber != 1 {
		t.Errorf("after white move, expected fullmove 1, got %d", Game.FullmoveNumber)
	}

	// Black moves e7-e5 → fullmove increments to 2.
	blackMove := types.Move{From: 52, To: 36, Flag: types.FlagDoublePush}
	Game.Make(blackMove)
	if Game.FullmoveNumber != 2 {
		t.Errorf("after black move, expected fullmove 2, got %d", Game.FullmoveNumber)
	}

	// Unmake black move → fullmove back to 1.
	Game.Unmake(blackMove)
	if Game.FullmoveNumber != 1 {
		t.Errorf("after unmake of black move, expected fullmove 1, got %d", Game.FullmoveNumber)
	}

	// Unmake white move → fullmove stays 1.
	Game.Unmake(whiteMove)
	if Game.FullmoveNumber != 1 {
		t.Errorf("after unmake of white move, expected fullmove 1, got %d", Game.FullmoveNumber)
	}
}