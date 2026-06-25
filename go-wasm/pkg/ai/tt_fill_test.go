package ai

import (
	"fmt"
	"testing"
	"webassemble/pkg/engine"
)

// TestTTFillMeasurement measures TT fill percentage across positions and time
// budgets. The TT should be ~40-50% full at 1s — if it's near 100%, the TT is
// too small and thrashes (always-replace would evict deep entries). Run with:
//
//	go test ./pkg/ai/ -v -run TestTTFillMeasurement -timeout 120s
func TestTTFillMeasurement(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TT fill measurement in short mode")
	}

	positions := []struct {
		name string
		fen  string
	}{
		{"Starting", engine.StartingFEN},
		{"Middlegame", "r1bqkbnr/pppp1ppp/2n5/1B2p3/4P3/5N2/PPPP1PPP/RNBQK2R w KQkq - 0 4"},
		{"Tactical", "r3k2r/p1p1qppp/2n5/3p4/3P4/2N5/PPPQ1PPP/R3K2R w KQkq - 0 1"},
		{"Endgame", "8/8/8/8/8/8/4k3/4K3 w - - 0 1"},
	}
	timeLimits := []int{500, 1000, 2000}

	t.Log("┌──────────────┬────────┬───────┬──────────┬────────┬────────────┐")
	t.Log("│ position     │ time   │ depth │ nodes    │ fill %  │ TT slots   │")
	t.Log("├──────────────┼────────┼───────┼──────────┼────────┼────────────┤")

	for _, pos := range positions {
		for _, limit := range timeLimits {
			engine.LoadFen(pos.fen)
			tt := engine.DefaultTranspositionTable()
			result := SearchWithTT(engine.Game, limit, nil, tt)

			t.Log(fmt.Sprintf("│ %-12s │ %4dms │  %2d   │ %8d │ %5.1f%% │ %10d │",
				pos.name, limit, result.Depth, result.Nodes, tt.FillPercent(), tt.Size()))
		}
	}

	t.Log("└──────────────┴────────┴───────┴──────────┴────────┴────────────┘")
}

// TestTTFillPerDepth shows at which depth the TT crosses 50/90/100% fill during
// iterative deepening on the starting position. This reveals when the TT starts
// thrashing and guides the ideal size. Run with:
//
//	go test ./pkg/ai/ -v -run TestTTFillPerDepth -timeout 120s
func TestTTFillPerDepth(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TT fill per-depth test in short mode")
	}

	t.Log("┌───────┬──────────┬────────┬────────────┐")
	t.Log("│ depth │ nodes    │ fill % │ TT slots   │")
	t.Log("├───────┼──────────┼────────┼────────────┤")

	for depth := 1; depth <= 14; depth++ {
		engine.LoadFen(engine.StartingFEN)
		tt := engine.DefaultTranspositionTable()
		result := SearchFixedDepthWithTT(engine.Game, depth, nil, tt)

		t.Log(fmt.Sprintf("│  %2d  │ %8d │ %5.1f%% │ %10d │",
			depth, result.Nodes, tt.FillPercent(), tt.Size()))
	}

	t.Log("└───────┴──────────┴────────┴────────────┘")
}

// TestTTPreservesDeepEntries verifies that gen-aware replacement keeps deep
// entries alive when a shallower store targets the same slot with a lower
// gen+depth priority. Two searches are run on the same TT: the first at a deep
// fixed depth, the second at a shallow fixed depth. The second search's shallow
// entries should NOT evict the first search's deep entries (same gen is not
// possible here since each Search call increments gen, so the shallow search
// has a higher gen — we verify that depth difference dominates).
//
// This is a sanity check, not a strict assertion — the exact behavior depends
// on collision rates. We log the fill before and after to observe the effect.
func TestTTPreservesDeepEntries(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TT preservation test in short mode")
	}

	engine.LoadFen(engine.StartingFEN)
	tt := engine.DefaultTranspositionTable()

	// Deep search first — fills TT with high-priority (gen=1, depth=10) entries.
	SearchFixedDepthWithTT(engine.Game, 10, nil, tt)
	fillAfterDeep := tt.FillPercent()
	t.Logf("after depth-10 search: fill=%.1f%%", fillAfterDeep)

	// Shallow search on a different position — its entries have gen=2 but
	// shallow depth (1-2). Gen+depth priority: gen=2+1=3 vs old gen=1+10=11.
	// The deep entries should survive most collisions.
	engine.LoadFen("r1bqkbnr/pppp1ppp/2n5/1B2p3/4P3/5N2/PPPP1PPP/RNBQK2R w KQkq - 0 4")
	SearchFixedDepthWithTT(engine.Game, 2, nil, tt)
	fillAfterShallow := tt.FillPercent()
	t.Logf("after depth-2 search:  fill=%.1f%%", fillAfterShallow)

	// The fill should not have grown much — shallow entries mostly rejected
	// because gen+depth priority is lower than the existing deep entries.
	growth := fillAfterShallow - fillAfterDeep
	t.Logf("fill growth from shallow search: %.1f%% (small = deep entries preserved)", growth)
}