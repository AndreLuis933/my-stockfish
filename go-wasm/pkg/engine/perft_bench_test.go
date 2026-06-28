package engine

import (
	"fmt"
	"testing"
	"time"
)

// TestPerftSpeed measures raw perft performance (move gen + make/unmake).
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
		LoadFen(tc.fen)
		start := time.Now()
		nodes := Game.Perft(tc.depth)
		elapsed := time.Since(start)
		nps := int64(nodes) * int64(time.Second) / int64(elapsed)
		fmt.Printf("%-25s: %d nodes in %v (%d Mnps)\n", tc.name, nodes, elapsed, nps/1_000_000)
		t.Logf("%-25s: %d nodes in %v (%d Mnps)", tc.name, nodes, elapsed, nps/1_000_000)
	}
}