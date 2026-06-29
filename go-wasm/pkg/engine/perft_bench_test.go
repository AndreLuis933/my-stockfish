package engine

import (
	"fmt"
	"testing"
	"time"
)

func runPerftBench(t *testing.T, name, fen string, depth int, fn func(int) int) {
	t.Helper()
	LoadFen(fen)
	start := time.Now()
	nodes := fn(depth)
	elapsed := time.Since(start)
	nps := int64(nodes) * int64(time.Second) / int64(elapsed)
	fmt.Printf("%-35s: %d nodes in %v (%d Mnps)\n", name, nodes, elapsed, nps/1_000_000)
	t.Logf("%-35s: %d nodes in %v (%d Mnps)", name, nodes, elapsed, nps/1_000_000)
}

// TestPerftSpeed compares the reference Perft against the lightweight PerftFast.
func TestPerftSpeed(t *testing.T) {
	fens := []struct {
		fen   string
		depth int
		name  string
	}{
		{StartingFEN, 5, "starting depth 5"},
		{"r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1", 4, "kiwipete depth 4"},
		{"r4rk1/1pp1qppp/p1np1n2/2b1p1B1/2B1P1b1/P1NP1N2/1PP1QPPP/R4RK1 w - - 0 10", 4, "position 6 depth 4"},
	}

	for _, tc := range fens {
		Game.reset()
		runPerftBench(t, tc.name+" (Perft)", tc.fen, tc.depth, Game.Perft)

		Game.reset()
		runPerftBench(t, tc.name+" (PerftFast)", tc.fen, tc.depth, Game.PerftFast)
	}
}