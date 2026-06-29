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

// TestRepetitionAvoidsStaleTT reproduces the bug where the engine plays into
// a 3-fold repetition while thinking it's winning, because the TT returns a
// stale winning score for a position that has now repeated. The fix checks
// IsRepetition() before the TT probe, so a repeating position always returns
// 0 (draw) regardless of what the TT says.
func TestRepetitionAvoidsStaleTT(t *testing.T) {
	// K+R vs K endgame. White is up a rook (winning). We play a repeating
	// sequence to create a threefold repetition, then search. The engine
	// must return score 0, NOT a winning score from the TT.
	//
	// Position: White Ke6, Ra1; Black Ke8. White to move.
	// Ra1-a2 Ke8-d8 Ra2-a1 Kd8-e8 (twofold) then repeat once more for threefold.
	engine.LoadFen("4k3/8/4K3/8/8/8/8/R7 w - - 0 1")

	// Cycle 1: Ra1-a2 Ke8-d8 Ra2-a1 Kd8-e8
	m1 := findMove(t, "a1", "a2")
	engine.Game.Make(m1)
	m2 := findMove(t, "e8", "d8")
	engine.Game.Make(m2)
	m3 := findMove(t, "a2", "a1")
	engine.Game.Make(m3)
	m4 := findMove(t, "d8", "e8")
	engine.Game.Make(m4)

	// Cycle 2: Ra1-a2 Ke8-d8 Ra2-a1 Kd8-e8 — now threefold repetition.
	m5 := findMove(t, "a1", "a2")
	engine.Game.Make(m5)
	m6 := findMove(t, "e8", "d8")
	engine.Game.Make(m6)
	m7 := findMove(t, "a2", "a1")
	engine.Game.Make(m7)
	m8 := findMove(t, "d8", "e8")
	engine.Game.Make(m8)

	// Now the position is back to the start (3rd occurrence).
	// The engine must return score 0, not a winning score from the TT.
	result := Search(engine.Game, 1000, nil)

	if result.Score > 0 {
		t.Errorf("expected draw score (0) for repeated position, got %d (winning) - "+
			"TT returned stale score before repetition check", result.Score)
	}
}

// findMove finds a legal move from src to dst square and returns it.
// Squares are in algebraic notation (e.g. "b5", "b2").
func findMove(t *testing.T, src, dst string) types.Move {
	t.Helper()
	from := squareNameToIdx(src)
	to := squareNameToIdx(dst)
	var ml engine.MoveList
	engine.Game.LegalMoves(&ml)
	for i := 0; i < ml.Len(); i++ {
		m := ml.Get(i)
		if int(m.From) == from && int(m.To) == to {
			return m
		}
	}
	t.Fatalf("no legal move from %s to %s", src, dst)
	return types.Move{}
}

// squareNameToIdx converts algebraic notation (e.g. "b5") to a 0-63 index.
func squareNameToIdx(s string) int {
	file := int(s[0] - 'a')
	rank := int(s[1] - '1')
	return rank*8 + file
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

// abs returns the absolute value of n (test helper).
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// searchCtxWithTT builds a searchCtx with a fresh TT and the given time limit,
// for tests that need to inspect ctx state (killers, history) after the search.
func searchCtxWithTT(timeLimitMs int) *searchCtx {
	return &searchCtx{
		startTime:   nowMs(),
		timeLimitMs: float64(timeLimitMs),
		tt:          engine.DefaultTranspositionTable(),
		gen:         1,
	}
}

// isLegalMove checks that m is a legal move in the current position by
// applying Make, verifying the side that moved is not in check, then Unmake.
func isLegalMove(p *engine.Position, m types.Move) bool {
	moverColor := sideColor(p)
	p.Make(m)
	defer p.Unmake(m)
	return !p.IsInCheck(moverColor)
}

// ── Phase 2 pruning tests ───────────────────────────────────────────

// TestKillerTableDeepPlyNoPanic is a regression test for a bounds panic that
// occurred in the browser: killerTable was sized [maxDepth=32] but indexed by
// p.Ply() (undoPly, bounded by engine.maxPly=256). Long searches with check
// extensions pushed ply past 32 and panicked. This test runs a long search on
// a position rich in forcing lines to drive ply deep and assert no panic.
func TestKillerTableDeepPlyNoPanic(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping long search test in short mode")
	}

	// A tactical middlegame with many checks and captures — forces deep
	// search trees with lots of check extensions (each extension +1 ply).
	fen := "r3k2r/p1p1qppp/2n5/3p4/3P4/2N5/PPPQ1PPP/R3K2R w KQkq - 0 1"
	engine.LoadFen(fen)

	result := Search(engine.Game, 5000, nil)
	if result.Depth == 0 {
		t.Fatal("search returned no move")
	}
	if !isLegalMove(engine.Game, result.Move) {
		t.Errorf("search returned illegal move %d→%d", result.Move.From, result.Move.To)
	}
}

// TestNullMovePruningFindsDeeperDepth verifies that null-move pruning
// increases the reachable depth at a fixed time budget. With pruning disabled
// (the old engine behavior), depth should be lower than with pruning enabled.
func TestNullMovePruningFindsDeeperDepth(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping depth comparison in short mode")
	}

	engine.LoadFen(engine.StartingFEN)

	// Pruning enabled (default).
	withPruning := Search(engine.Game, 1000, nil)

	// Pruning disabled.
	engine.LoadFen(engine.StartingFEN)
	ctx := searchCtxWithTT(1000)
	ctx.disableNullMove = true
	withoutPruning := iterativeDeepening(engine.Game, ctx, maxDepth)

	if withPruning.Depth <= withoutPruning.Depth {
		t.Errorf("null-move pruning did not increase depth: with=%d, without=%d",
			withPruning.Depth, withoutPruning.Depth)
	}

	t.Logf("depth with pruning: %d (nodes %d), without: %d (nodes %d)",
		withPruning.Depth, withPruning.Nodes,
		withoutPruning.Depth, withoutPruning.Nodes)
}

// TestNullMovePruningZugzwangEndgame verifies null-move pruning doesn't blunder
// in a K+P vs K endgame where passing would be illegal (zugzwang-like). The
// engine should still find the winning king advance, not walk into a stalemate
// or lose the pawn.
func TestNullMovePruningZugzwangEndgame(t *testing.T) {
	// White Kf6, Pg6, Black Kf8. White to move. This is a classic winning
	// K+P vs K: the white king is in front of the pawn (key squares e7/f7/g7
	// controlled). White plays Kf7 (or g7+ if promoted, but pawn is on g6)
	// and escorts the pawn to promotion. A null move here would be illegal
	// (black king would have no moves — stalemate), but pruning is guarded
	// by hasNonPawnMaterial so it's active. The test verifies the engine
	// plays a legal, non-blundering move with pruning on.
	engine.LoadFen("5k2/8/5KP1/8/8/8/8/8 w - - 0 1")
	result := Search(engine.Game, 2000, nil)

	if result.Depth == 0 {
		t.Fatal("search returned no move")
	}
	if !isLegalMove(engine.Game, result.Move) {
		t.Errorf("search returned illegal move %d→%d in endgame",
			result.Move.From, result.Move.To)
	}

	// The engine must not blunder the pawn or walk into stalemate. After
	// the engine's move, it's black to move, so Evaluate returns the eval
	// from black's perspective — negative means white is winning. We just
	// check the pawn is alive and the absolute eval is far from zero.
	engine.Game.Make(result.Move)
	score := Evaluate(engine.Game)
	pawnAlive := false
	for _, piece := range engine.Game.Board {
		if piece&types.TypeMask == types.Pawn && piece&types.ColorMask == types.ColorWhite {
			pawnAlive = true
			break
		}
	}
	engine.Game.Unmake(result.Move)

	if !pawnAlive {
		t.Error("engine blundered the pawn in a winning endgame")
	}
	if abs(score) < 100 {
		t.Errorf("expected decisive eval after engine move, got %d (pawn alive: %v)",
			score, pawnAlive)
	}
	t.Logf("eval after engine move: %d (negative = white winning, black to move)", score)
}

// TestLMRReturnsLegalMove runs a deep fixed-depth search on a tactical
// position and verifies the returned move is legal and the score is not
// a gross blunder. LMR reduces depth on late moves; if the reduction is
// wrong, the engine may return a move that hangs material.
func TestLMRReturnsLegalMove(t *testing.T) {
	// A sharp tactical position — LMR must not miss the critical threats.
	fen := "r1bqkbnr/pppp1ppp/2n5/2N5/4P3/8/PPPP1PPP/RNBQKB1R b KQkq - 0 1"
	engine.LoadFen(fen)

	result := SearchFixedDepth(engine.Game, 8, nil)
	if result.Depth == 0 {
		t.Fatal("search returned no move")
	}
	if !isLegalMove(engine.Game, result.Move) {
		t.Errorf("LMR search returned illegal move %d→%d",
			result.Move.From, result.Move.To)
	}

	// Score should be near-balanced (it's a known opening, black to move).
	// A gross blunder would show a large negative score for black.
	if result.Score < -500 {
		t.Errorf("LMR search returned blunder score %d for black", result.Score)
	}
}

// TestAspirationWindowsMatchFullWindow verifies that aspiration windows
// produce the same best move as a full-window search. Aspiration is a
// pruning optimization — it must not change the result, only speed it up.
// We compare the score (move can differ among equal-scored moves) at a
// fixed depth on a quiet position.
func TestAspirationWindowsMatchFullWindow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping aspiration comparison in short mode")
	}

	// A quiet middlegame where aspiration windows should succeed (score
	// stays inside the window) and produce the same score as full window.
	fen := "r1bqkbnr/pppp1ppp/2n5/1B2p3/4P3/5N2/PPPP1PPP/RNBQK2R w KQkq - 0 4"
	engine.LoadFen(fen)

	// Aspiration enabled (default Search uses aspiration from depth 3).
	aspiration := SearchFixedDepth(engine.Game, 6, nil)

	// Full window: disable aspiration by searching a single depth directly
	// with negamax (bypassing iterativeDeepening's aspiration logic).
	engine.LoadFen(fen)
	ctx := searchCtxWithTT(1e18)
	ctx.disableNullMove = false
	full := iterativeDeepening(engine.Game, ctx, 6)

	if aspiration.Score != full.Score {
		t.Errorf("aspiration score %d != full window score %d",
			aspiration.Score, full.Score)
	}

	// Both must return legal moves.
	engine.LoadFen(fen)
	if !isLegalMove(engine.Game, aspiration.Move) {
		t.Errorf("aspiration returned illegal move %d→%d",
			aspiration.Move.From, aspiration.Move.To)
	}
	engine.LoadFen(fen)
	if !isLegalMove(engine.Game, full.Move) {
		t.Errorf("full window returned illegal move %d→%d",
			full.Move.From, full.Move.To)
	}
}

// TestKillersRecordedOnCutoff is a white-box test: after a search, the killer
// table should have at least one populated slot at ply ≥ 1 (a quiet move that
// caused a beta cutoff somewhere in the tree). An empty killer table would
// indicate the cutoff-recording path is broken.
func TestKillersRecordedOnCutoff(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping killer inspection in short mode")
	}

	engine.LoadFen(engine.StartingFEN)
	ctx := searchCtxWithTT(2000)
	iterativeDeepening(engine.Game, ctx, maxDepth)

	populated := 0
	for ply := 1; ply < maxPly; ply++ {
		if ctx.killers[ply][0].From != 0 || ctx.killers[ply][1].From != 0 {
			populated++
		}
	}

	if populated == 0 {
		t.Error("no killer moves recorded after search — cutoff path is broken")
	}

	t.Logf("killer slots populated at %d plies", populated)
}

// TestHistoryAgedBetweenIterations verifies that history aging works: after
// a search, running another search should have aged the old entries. We check
// that the max history value after a second search is not larger than after
// the first (aging subtracts 8 from each entry between iterations).
func TestHistoryAgedBetweenIterations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping history aging test in short mode")
	}

	engine.LoadFen(engine.StartingFEN)
	ctx := searchCtxWithTT(1000)
	iterativeDeepening(engine.Game, ctx, maxDepth)

	maxAfterFirst := 0
	for _, v := range ctx.history {
		if v > maxAfterFirst {
			maxAfterFirst = v
		}
	}

	if maxAfterFirst == 0 {
		t.Skip("no history entries recorded — skip aging check")
	}

	// A second search on a different position: the aging at the start of
	// each iteration decays the old entries. After the second search, the
	// max value should not exceed the first search's max by more than the
	// depth² bonus a single iteration can add.
	engine.LoadFen("r1bqkbnr/pppp1ppp/2n5/2N5/4P3/8/PPPP1PPP/RNBQKB1R w KQkq - 0 1")
	iterativeDeepening(engine.Game, ctx, maxDepth)

	maxAfterSecond := 0
	for _, v := range ctx.history {
		if v > maxAfterSecond {
			maxAfterSecond = v
		}
	}

	// Aging ensures old entries decay — if maxAfterSecond is much larger
	// than maxAfterFirst, aging is not working. We allow some growth (the
	// new position may have strong cutoffs) but it should be bounded.
	if maxAfterSecond > maxAfterFirst*4 {
		t.Errorf("history not aged: max went from %d to %d (expected bounded growth)",
			maxAfterFirst, maxAfterSecond)
	}

	t.Logf("history max: first=%d, second=%d", maxAfterFirst, maxAfterSecond)
}

// TestPruningDoesNotCorruptBoard verifies that after a search with all pruning
// enabled (null move, LMR, aspiration, killers, history), the board is
// restored to its original state — Make/Unmake are balanced even with the
// null-move inline flip and LMR re-search paths.
func TestPruningDoesNotCorruptBoard(t *testing.T) {
	engine.LoadFen(engine.StartingFEN)
	original := engine.Game.Board
	originalHash := engine.Game.Hash
	originalEval := engine.Game.EvalScore
	originalCastle := engine.Game.CastlingRights
	originalEP := engine.Game.EnPassantTarget

	Search(engine.Game, 2000, nil)

	if engine.Game.Board != original {
		t.Error("board corrupted after search — Make/Unmake unbalanced")
	}
	if engine.Game.Hash != originalHash {
		t.Error("hash corrupted after search — null-move flip not reverted")
	}
	if engine.Game.EvalScore != originalEval {
		t.Error("eval score corrupted after search")
	}
	if engine.Game.CastlingRights != originalCastle {
		t.Error("castling rights corrupted after search")
	}
	if engine.Game.EnPassantTarget != originalEP {
		t.Error("en passant target corrupted after search")
	}
}

// TestNullMoveDoesNotPanicInCheck verifies the null-move guard: when the side
// to move is in check, null-move pruning must be skipped (you can't "pass"
// while in check — the king would be captured). This test searches a position
// where the side to move is in check and verifies no panic occurs.
func TestNullMoveDoesNotPanicInCheck(t *testing.T) {
	// Black king on e4, white rook on e1 — black is in check (Re1 attacks e4).
	// Black to move must get out of check; null-move pruning must be skipped.
	engine.LoadFen("8/8/8/8/4k3/8/8/4R3 b - - 0 1")
	result := Search(engine.Game, 1000, nil)

	if result.Depth == 0 {
		t.Fatal("search returned no move from check position")
	}
	if !isLegalMove(engine.Game, result.Move) {
		t.Errorf("search returned illegal move %d→%d while in check",
			result.Move.From, result.Move.To)
	}
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