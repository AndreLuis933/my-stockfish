package ai

import (
	"fmt"
	"testing"
	"time"
	"webassemble/pkg/engine"
	"webassemble/pkg/types"
)

// ── Helpers ─────────────────────────────────────────────────────────

func searchBest(t *testing.T, fen string, timeLimitMs int) SearchResult {
	t.Helper()
	engine.LoadFen(fen)
	result := Search(engine.Game, timeLimitMs, nil)
	if result.Depth == 0 {
		t.Fatal("AI returned no move (depth 0)")
	}
	return result
}

// isMate checks if making the AI's move results in checkmate for the opponent.
func isMate(move types.Move) bool {
	engine.Game.Make(move)
	defer engine.Game.Unmake(move)
	status := engine.Game.CurrentStatus()
	return status == engine.StatusWhiteWins || status == engine.StatusBlackWins
}

// ── Evaluation tests ────────────────────────────────────────────────

func TestEvaluateStartingPosition(t *testing.T) {
	engine.LoadFen(engine.StartingFEN)
	score := Evaluate(engine.Game)
	if score != 0 {
		t.Errorf("expected symmetric position to evaluate to 0, got %d", score)
	}
}

func TestEvaluateMaterialImbalance(t *testing.T) {
	engine.LoadFen("rnbqkbnr/pppp1ppp/8/4p3/6P1/5P2/PPPPP2P/RNBQKBNR b KQkq - 0 2")
	score := Evaluate(engine.Game)
	if score <= 0 {
		t.Errorf("black is up a pawn (captured), expected positive score for black to move, got %d", score)
	}
}

func TestEvaluateUpMaterialWhenAhead(t *testing.T) {
	engine.LoadFen("8/8/8/8/8/8/PPPPPPPP/4K3 w - - 0 1")
	score := Evaluate(engine.Game)
	if score < 500 {
		t.Errorf("white is up a queen worth of pawns, expected high score, got %d", score)
	}
}

func TestSideColor(t *testing.T) {
	engine.LoadFen(engine.StartingFEN)
	if sideColor(engine.Game) != types.ColorWhite {
		t.Error("expected white to move at start")
	}
	engine.LoadFen("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR b KQkq - 0 1")
	if sideColor(engine.Game) != types.ColorBlack {
		t.Error("expected black to move")
	}
}

// ── Mate tests ──────────────────────────────────────────────────────
// Each test verifies the AI can find a forced mate within the given time.
// The FEN must lead to a forced mate in exactly N moves.

func TestMateIn1(t *testing.T) {
	// Back-rank mate: white rook on e1, Re8# (black king on g8, pawns block escape).
	result := searchBest(t, "6k1/5ppp/8/8/8/8/8/4R1K1 w - - 0 1", 2000)
	if !isMate(result.Move) {
		t.Errorf("expected checkmate, got status after move %d→%d",
			result.Move.From, result.Move.To)
	}
}

func TestMateIn1Black(t *testing.T) {
	// Back-rank mate for black: rook on e8, white king on g1, pawns f2/g2/h2 block escape.
	// Black plays Re8-e1# (rook from index 60 to index 4).
	result := searchBest(t, "4r1k1/8/8/8/8/8/5PPP/6K1 b - - 0 1", 2000)
	if !isMate(result.Move) {
		t.Errorf("expected checkmate, got status after move %d→%d",
			result.Move.From, result.Move.To)
	}
}

// TestMateIn1WithHighHalfmoveClock reproduces a bug from a real game (pgn.txt):
// in a K+Q vs K endgame, the halfmove clock exceeded 100 and the engine
// returned 0 (draw) at the root before detecting mate-in-1 was available.
// Checkmate must be detected even when the 50-move rule clock is high.
func TestMateIn1WithHighHalfmoveClock(t *testing.T) {
	// White Ke6, Qg7; Black Ke8. Qg8# — king can't flee (all escapes
	// covered by Ke6 and Qg8). Halfmove clock set to 120 (past 50-move rule).
	result := searchBest(t, "4k3/6Q1/4K3/8/8/8/8/8 w - - 120 1", 2000)
	if !isMate(result.Move) {
		t.Errorf("expected checkmate with high halfmove clock, got move %d→%d",
			result.Move.From, result.Move.To)
	}
}

// TestMateIn1RepetitionInSearchTree reproduces the pgn.txt bug where the
// engine failed to find mate-in-1 in a K+Q vs K endgame because the position
// after the mating move had occurred 3× in the search path (threefold
// repetition), causing IsRepetition() to return 0 (draw) before the move loop
// could detect that the side to move had no legal moves (checkmate).
func TestMateIn1RepetitionInSearchTree(t *testing.T) {
	// White Kc3, Qb5; Black Ka1. Mate-in-1: Qb5-b2# (defended by Kc3, which
	// is diagonally adjacent to b2). This is the exact position from move
	// 120 of the pgn.txt game.
	result := searchBest(t, "8/8/8/1Q6/8/2K5/8/k7 w - - 0 1", 2000)
	if !isMate(result.Move) {
		t.Errorf("expected Qb2# checkmate, got move %d→%d",
			result.Move.From, result.Move.To)
	}
}

func TestWinsHangingPiece(t *testing.T) {
	// White knight on c5 is undefended; black to move should capture it with the bishop.
	// After 1...Bxc5, black is up a knight (320 points).
	engine.LoadFen("r1bqkbnr/pppp1ppp/2n5/2N5/8/8/PPPPPPPP/R1BQKBNR b KQkq - 0 1")
	result := Search(engine.Game, 2000, nil)

	if result.Depth == 0 {
		t.Fatal("AI returned no move")
	}

	// The AI should capture the knight on c5 (index 34)
	if result.Move.To != 34 {
		// Maybe the AI found something better, but capturing a free knight is clearly winning
		// Check the score is positive for black
		if result.Score < 200 {
			t.Errorf("expected to win material (score > 200), got score %d, move %d→%d",
				result.Score, result.Move.From, result.Move.To)
		}
	}
}

// ── Search properties tests ─────────────────────────────────────────

func TestSearchReturnsValidMove(t *testing.T) {
	result := searchBest(t, engine.StartingFEN, 500)

	var ml engine.MoveList
	engine.Game.PseudoLegalMoves(&ml)
	moverColor := sideColor(engine.Game)

	found := false
	for i := 0; i < ml.Len(); i++ {
		m := ml.Get(i)
		if m.From == result.Move.From && m.To == result.Move.To && m.Promotion == result.Move.Promotion {
			engine.Game.Make(m)
			if !engine.Game.IsInCheck(moverColor) {
				found = true
			}
			engine.Game.Unmake(m)
			break
		}
	}
	if !found {
		t.Errorf("AI returned illegal move: %d→%d", result.Move.From, result.Move.To)
	}
}

func TestSearchFindsCheckmateScore(t *testing.T) {
	result := searchBest(t, "6k1/5ppp/8/8/8/8/8/4R1K1 w - - 0 1", 2000)
	if result.Score < winScore/2 {
		t.Errorf("expected mate score (≥%d), got %d", winScore/2, result.Score)
	}
}

func TestSearchDoesNotHang(t *testing.T) {
	result := searchBest(t, engine.StartingFEN, 100)
	if result.TimeMs > 500 {
		t.Errorf("search exceeded time budget significantly: %dms (limit 100ms)", result.TimeMs)
	}
}

func TestNodeTypeConsistency(t *testing.T) {
	engine.LoadFen(engine.StartingFEN)
	original := engine.Game.Board

	Search(engine.Game, 300, nil)

	for i, piece := range engine.Game.Board {
		if piece != original[i] {
			t.Errorf("board modified at %d: was %d now %d — Make/Unmake not balanced", i, original[i], piece)
		}
	}
}

func TestSearchNodesPositive(t *testing.T) {
	result := searchBest(t, engine.StartingFEN, 200)
	if result.Nodes <= 0 {
		t.Errorf("expected positive node count, got %d", result.Nodes)
	}
}

// ── Depth / time / nodes measurements ────────────────────────────────
// These tests don't assert pass/fail — they print measurements so you can
// see how depth scales with time. Run with: go test -v -run TestDepthScaling

func TestDepthScaling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping depth scaling test in short mode")
	}

	timeLimits := []int{100, 500, 1000, 2000, 5000}

	t.Log("┌─────────────┬───────┬────────┬──────────┬───────────┐")
	t.Log("│ time (ms)   │ depth │ nodes  │ time (ms) │ nodes/sec │")
	t.Log("├─────────────┼───────┼────────┼──────────┼───────────┤")

	for _, limit := range timeLimits {
		engine.LoadFen(engine.StartingFEN)
		result := Search(engine.Game, limit, nil)

		nps := int64(0)
		if result.TimeMs > 0 {
			nps = int64(result.Nodes) * 1000 / result.TimeMs
		}

		t.Log(fmt.Sprintf("│ %9d   │  %2d   │ %6d │ %8d │ %9d │",
			limit, result.Depth, result.Nodes, result.TimeMs, nps))
	}

	t.Log("└─────────────┴───────┴────────┴──────────┴───────────┘")
}

func TestDepthScalingPerPosition(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	positions := []struct {
		name string
		fen  string
	}{
		{"Starting", engine.StartingFEN},
		{"Middlegame", "r1bqkbnr/pppp1ppp/2n5/1B2p3/4P3/5N2/PPPP1PPP/RNBQK2R w KQkq - 0 4"},
		{"Endgame", "8/8/8/8/8/8/4k3/4K3 w - - 0 1"},
		{"Tactical", "r3k2r/p1p1qppp/2n5/3p4/3P4/2N5/PPPQ1PPP/R3K2R w KQkq - 0 1"},
	}

	timeLimit := 2000

	t.Log("┌──────────────┬───────┬────────┬──────────┬───────────┐")
	t.Log("│ position     │ depth │ nodes  │ time (ms) │ nodes/sec │")
	t.Log("├──────────────┼───────┼────────┼──────────┼───────────┤")

	for _, pos := range positions {
		engine.LoadFen(pos.fen)
		result := Search(engine.Game, timeLimit, nil)

		nps := int64(0)
		if result.TimeMs > 0 {
			nps = int64(result.Nodes) * 1000 / result.TimeMs
		}

		t.Log(fmt.Sprintf("│ %-10s  │  %2d   │ %6d │ %8d │ %9d │",
			pos.name, result.Depth, result.Nodes, result.TimeMs, nps))
	}

	t.Log("└──────────────┴───────┴────────┴──────────┴───────────┘")
}

// ── Benchmarks ─────────────────────────────────────────────────────
// Run with: go test -bench=. -benchmem

func BenchmarkSearchDepth1(b *testing.B) {
	engine.LoadFen(engine.StartingFEN)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.LoadFen(engine.StartingFEN)
		Search(engine.Game, 100000, nil) // very high limit — depth 1 finishes instantly
	}
}

func BenchmarkSearchStartingPosition(b *testing.B) {
	engine.LoadFen(engine.StartingFEN)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.LoadFen(engine.StartingFEN)
		Search(engine.Game, 1000, nil)
	}
}

func BenchmarkEvaluate(b *testing.B) {
	engine.LoadFen(engine.StartingFEN)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Evaluate(engine.Game)
	}
}

func BenchmarkPerftDepth4(b *testing.B) {
	engine.LoadFen(engine.StartingFEN)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.LoadFen(engine.StartingFEN)
		engine.Game.Perft(4)
	}
}

// TestFixedDepthMeasuresNPS runs a fixed-depth search (no time limit) and
// reports nodes/sec. This gives a clean performance number without
// time-limit interference. Run with: go test -v -run TestFixedDepthMeasuresNPS
func TestFixedDepthMeasuresNPS(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping nps test in short mode")
	}

	depths := []int{1, 2, 3, 4, 5}

	t.Log("┌───────┬──────────┬──────────┬───────────┐")
	t.Log("│ depth │ nodes    │ time(ms) │ nodes/sec │")
	t.Log("├───────┼──────────┼──────────┼───────────┤")

	for _, depth := range depths {
		engine.LoadFen(engine.StartingFEN)

		start := time.Now()
		result := searchFixedDepth(engine.Game, depth)
		elapsed := time.Since(start).Milliseconds()

		nps := int64(0)
		if elapsed > 0 {
			nps = int64(result.Nodes) * 1000 / elapsed
		}

		t.Log(fmt.Sprintf("│  %2d   │ %8d │ %8d │ %9d │",
			depth, result.Nodes, elapsed, nps))
	}

	t.Log("└───────┴──────────┴──────────┴───────────┘")
}

// searchFixedDepth runs a fixed-depth search with no time limit.
func searchFixedDepth(p *engine.Position, depth int) SearchResult {
	return SearchFixedDepth(p, depth, nil)
}

// TestIterativeVsDirect compares the node count of iterative deepening to
// depth N vs a direct search at depth N. ID searches depths 1..N, so it does
// extra work at shallow depths — but the previousBest hint improves cutoffs
// at the target depth, so it may recover some of that cost.
func TestIterativeVsDirect(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	depths := []int{1, 2, 3, 4, 5, 6}

	t.Log("┌───────┬──────────────┬──────────────┬───────────┐")
	t.Log("│ depth │ ID nodes     │ direct nodes │ ID/direct │")
	t.Log("├───────┼──────────────┼──────────────┼───────────┤")

	for _, depth := range depths {
		// ID to `depth`: huge time limit so it won't abort, depth cap = depth.
		engine.LoadFen(engine.StartingFEN)
		ctx := &searchCtx{startTime: nowMs(), timeLimitMs: 1e18}
		idResult := iterativeDeepening(engine.Game, ctx, depth)

		engine.LoadFen(engine.StartingFEN)
		directResult := SearchFixedDepth(engine.Game, depth, nil)

		ratio := float64(0)
		if directResult.Nodes > 0 {
			ratio = float64(idResult.Nodes) / float64(directResult.Nodes)
		}

		t.Log(fmt.Sprintf("│  %2d   │ %12d │ %12d │    %.2f   │",
			depth, idResult.Nodes, directResult.Nodes, ratio))
	}

	t.Log("└───────┴──────────────┴──────────────┴───────────┘")
}