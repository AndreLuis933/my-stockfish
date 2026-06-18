package engine

import "testing"

// Perft validation against the 6 standard positions from
// https://www.chessprogramming.org/Perft_Results
//
// Each case lists the FEN and the known node counts per depth.
// Depths are kept shallow enough to run fast in CI (under ~1s total).

func TestPerftInitialPosition(t *testing.T) {
	fen := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	want := map[int]int{
		1: 20,
		2: 400,
		3: 8902,
		4: 197281,
	}
	runPerft(t, fen, want)
}

func TestPerftKiwipete(t *testing.T) {
	// "Kiwipete" — exercises castling, en passant, promotions, checks.
	fen := "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - "
	want := map[int]int{
		1: 48,
		2: 2039,
		3: 97862,
	}
	runPerft(t, fen, want)
}

func TestPerftPosition3(t *testing.T) {
	// Position 3 — en passant, promotions, checks. Tricky edge cases.
	fen := "8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1"
	want := map[int]int{
		1: 14,
		2: 191,
		3: 2812,
		4: 43238,
	}
	runPerft(t, fen, want)
}

func TestPerftPosition4(t *testing.T) {
	// Position 4 — many promotions and castling edge cases.
	fen := "r3k2r/Pppp1ppp/1b3nbN/nP6/BBP1P3/q4N2/Pp1P2PP/R2Q1RK1 w kq - 0 1"
	want := map[int]int{
		1: 6,
		2: 264,
		3: 9467,
	}
	runPerft(t, fen, want)
}

func TestPerftPosition5(t *testing.T) {
	// Position 5 — known to catch bugs in engines years old at depth 3.
	fen := "rnbq1k1r/pp1Pbppp/2p5/8/2B5/8/PPP1NnPP/RNBQK2R w KQ - 1 8"
	want := map[int]int{
		1: 44,
		2: 1486,
		3: 62379,
	}
	runPerft(t, fen, want)
}

func TestPerftPosition6(t *testing.T) {
	// Position 6 — alternative perft, Steven Edwards.
	fen := "r4rk1/1pp1qppp/p1np1n2/2b1p1B1/2B1P1b1/P1NP1N2/1PP1QPPP/R4RK1 w - - 0 10"
	want := map[int]int{
		1: 46,
		2: 2079,
		3: 89890,
	}
	runPerft(t, fen, want)
}

func runPerft(t *testing.T, fen string, want map[int]int) {
	t.Helper()
	Game.reset()
	LoadFen(fen)
	for depth, expected := range want {
		got := Game.Perft(depth)
		if got != expected {
			t.Errorf("perft(%d) = %d, want %d (fen: %s)", depth, got, expected, fen)
		}
	}
}