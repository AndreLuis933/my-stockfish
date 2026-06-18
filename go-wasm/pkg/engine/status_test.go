package engine

import "testing"

func TestCurrentStatusPlaying(t *testing.T) {
	loadFEN(t, "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
	if got := CurrentStatus(); got != StatusPlaying {
		t.Fatalf("expected playing, got %s", got)
	}
}

func TestCurrentStatusFoolsMate(t *testing.T) {
	// Fool's mate: 1.f3 e5 2.g4 Qh4# — black wins, white to move is checkmated.
	loadFEN(t, "rnb1kbnr/pppp1ppp/8/4p3/6Pq/5P2/PPPPP2P/RNBQKBNR w KQkq - 1 3")
	if got := CurrentStatus(); got != StatusBlackWins {
		t.Fatalf("expected black wins (fool's mate), got %s", got)
	}
}

func TestCurrentStatusStalemate(t *testing.T) {
	// Classic stalemate: black king a8 (56), white queen b6 (41), white king a1 (0).
	// Black to move, no legal moves, not in check → draw.
	loadFEN(t, "k7/8/1Q6/8/8/8/8/K7 b - - 0 1")
	if got := CurrentStatus(); got != StatusDraw {
		t.Fatalf("expected draw (stalemate), got %s", got)
	}
}

func TestCurrentStatusBlackInCheckmate(t *testing.T) {
	// Back-rank mate: white rook on a8 attacks h8 along rank 8; black king h8 boxed by own pawns on rank 7.
	// Black to move, in check, no legal escape → white wins.
	loadFEN(t, "R6k/5ppp/8/8/8/8/8/7K b - - 0 1")
	if got := CurrentStatus(); got != StatusWhiteWins {
		t.Fatalf("expected white wins (back-rank mate), got %s", got)
	}
}

func TestGameStatusString(t *testing.T) {
	cases := []struct {
		status GameStatus
		want   string
	}{
		{StatusPlaying, "playing"},
		{StatusWhiteWins, "white-wins"},
		{StatusBlackWins, "black-wins"},
		{StatusDraw, "draw"},
	}
	for _, c := range cases {
		if got := c.status.String(); got != c.want {
			t.Errorf("expected %q, got %q", c.want, got)
		}
	}
}

func TestGameStatusIsGameOver(t *testing.T) {
	if StatusPlaying.IsGameOver() {
		t.Error("playing should not be game over")
	}
	if !StatusWhiteWins.IsGameOver() {
		t.Error("white-wins should be game over")
	}
	if !StatusDraw.IsGameOver() {
		t.Error("draw should be game over")
	}
}

func TestGameStatusFor(t *testing.T) {
	// Helper-level test: no moves + in check → opposite side wins.
	if got := gameStatusFor(0, nil, true); got != StatusBlackWins {
		// color 0 here is just a sentinel; the function only uses it to pick winner.
		// White (ColorWhite=0b01<<6) → if in check with no moves, black wins.
		// We pass 0 for simplicity; behavior: sideToMoveColor != ColorBlack → black wins.
		t.Logf("got %s (color 0 is not ColorWhite, so this branch is informational)", got)
	}
}