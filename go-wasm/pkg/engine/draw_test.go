package engine

import (
	"testing"
)

func TestThreefoldRepetition(t *testing.T) {
	var p Position
	p.LoadFen(StartingFEN)

	// Move knights back and forth to create a repetition:
	// 1. Nc3 Nc6  2. Nb1 Nb8  3. Nc3 Nc6 — position after 3.Nc3 is 3-fold.
	moveSeq := [][2]int{
		{1, 18},  // Nc3
		{57, 42}, // Nc6
		{18, 1},  // Nb1
		{42, 57}, // Nb8
		{1, 18},  // Nc3 — back to start
		{57, 42}, // Nc6
		{18, 1},  // Nb1
		{42, 57}, // Nb8
		{1, 18},  // Nc3 — third occurrence
	}

	for i, mv := range moveSeq {
		var ml MoveList
		p.LegalMoves(&ml)
		for j := 0; j < ml.Len(); j++ {
			m := ml.Get(j)
			if int(m.From) == mv[0] && int(m.To) == mv[1] {
				p.Make(m)
				break
			}
		}
		// After move 8 (0-indexed), we should have threefold.
		if i == 8 {
			if !p.IsRepetition() {
				t.Errorf("expected threefold repetition after move %d", i)
			}
		}
	}
}

func TestNoRepetition(t *testing.T) {
	var p Position
	p.LoadFen(StartingFEN)
	if p.IsRepetition() {
		t.Error("starting position should not be a repetition")
	}

	// One move — no repetition.
	var ml MoveList
	p.LegalMoves(&ml)
	p.Make(ml.Get(0))
	if p.IsRepetition() {
		t.Error("position after one move should not be a repetition")
	}
}

func TestInsufficientMaterialKvK(t *testing.T) {
	var p Position
	p.LoadFen("8/8/4k3/8/8/4K3/8/8 w - - 0 1")
	if !p.IsInsufficientMaterial() {
		t.Error("K vs K should be insufficient material")
	}
	if !p.IsDraw() {
		t.Error("K vs K should be a draw")
	}
}

func TestInsufficientMaterialKBvK(t *testing.T) {
	var p Position
	p.LoadFen("8/8/4k3/8/8/4KB2/8/8 w - - 0 1")
	if !p.IsInsufficientMaterial() {
		t.Error("K+B vs K should be insufficient material")
	}
}

func TestInsufficientMaterialKNvK(t *testing.T) {
	var p Position
	p.LoadFen("8/8/4k3/8/8/4KN2/8/8 w - - 0 1")
	if !p.IsInsufficientMaterial() {
		t.Error("K+N vs K should be insufficient material")
	}
}

func TestInsufficientMaterialKBvsKBsameColor(t *testing.T) {
	// Both bishops on light squares (c1=dark, f6=light... let me pick two same-color)
	// a1=0 (dark), c3=18 (light). Let me use e1 and e8:
	// e1=4 (file 4, rank 0 → 4+0=4, even=light), e8=60 (file 4, rank 7 → 4+7=11, odd=dark)
	// Need same color: c1=2 (2+0=2, even=light), f6=45 (5+5=10, even=light)
	var p Position
	p.LoadFen("5b2/8/4k3/8/8/4K3/8/2B5 w - - 0 1")
	if !p.IsInsufficientMaterial() {
		t.Error("K+B vs K+B with same-color bishops should be insufficient")
	}
}

func TestInsufficientMaterialKBvsKBdiffColor(t *testing.T) {
	// c1=2 (light), f6=45 (light) — same color, need different
	// a1=0 (dark), h8=63 (dark+light? 7+7=14, even=light)
	// a1=0 (0+0=0, even=light), a8=56 (0+7=7, odd=dark) — different!
	var p Position
	p.LoadFen("b7/8/4k3/8/8/4K3/8/B7 w - - 0 1")
	// a8=56 (file 0, rank 7, 0+7=7 odd=dark), a1=0 (0+0=0 even=light) — different
	if p.IsInsufficientMaterial() {
		t.Error("K+B vs K+B with different-color bishops should NOT be insufficient")
	}
}

func TestSufficientMaterialPawn(t *testing.T) {
	var p Position
	p.LoadFen("8/8/4k3/8/4P3/4K3/8/8 w - - 0 1")
	if p.IsInsufficientMaterial() {
		t.Error("K+P vs K should not be insufficient material")
	}
}

func TestSufficientMaterialRook(t *testing.T) {
	var p Position
	p.LoadFen("8/8/4k3/8/8/4K3/8/R7 w - - 0 1")
	if p.IsInsufficientMaterial() {
		t.Error("K+R vs K should not be insufficient material")
	}
}

func TestFiftyMoveRule(t *testing.T) {
	var p Position
	p.LoadFen("8/8/4k3/8/8/4K3/8/8 w - - 99 1")
	if p.IsFiftyMoveRule() {
		t.Error("99 halfmoves should not trigger 50-move rule")
	}
	p.HalfmoveClock = 100
	if !p.IsFiftyMoveRule() {
		t.Error("100 halfmoves should trigger 50-move rule")
	}
}

func TestCurrentStatusDraw(t *testing.T) {
	var p Position
	p.LoadFen("8/8/4k3/8/8/4K3/8/8 w - - 0 1")
	status := p.CurrentStatus()
	if status != StatusDraw {
		t.Errorf("K vs K should be StatusDraw, got %s", status)
	}
}