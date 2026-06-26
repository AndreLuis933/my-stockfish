package engine

import (
	"testing"

	"webassemble/pkg/types"
)

// Engine board layout: index 0 = a1, index 63 = h8.
// a1=0, h1=7, a2=8, e2=12, e4=28, g1=6, f3=21, b1=1, c3=18
// f1=5, d1=3, e1=4, a8=56, h8=63, e8=60, a7=48, a1=0

func TestToSan_StartingPosition(t *testing.T) {
	LoadFen(StartingFEN)
	tests := []struct {
		name string
		from int
		to   int
		want string
	}{
		{"pawn e2-e4", 12, 28, "e4"},
		{"knight g1-f3", 6, 21, "Nf3"},
		{"knight b1-c3", 1, 18, "Nc3"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := types.Move{From: tc.from, To: tc.to}
			piece := Game.Board[tc.from]
			if piece&types.Pawn != 0 && abs(tc.to-tc.from) == 2*boardSize {
				m.Flag = types.FlagDoublePush
			}
			san, err := Game.ToSan(m)
			if err != nil {
				t.Fatalf("ToSan error: %v", err)
			}
			if san != tc.want {
				t.Errorf("ToSan = %q, want %q", san, tc.want)
			}
		})
	}
}

func TestToSan_OpenPosition(t *testing.T) {
	// Position with no pawns on rank 2 so bishops/queen/king can move
	LoadFen("4k3/8/8/8/8/8/8/RNBQKBNR w - - 0 1")
	tests := []struct {
		name string
		from int
		to   int
		want string
	}{
		{"bishop f1-e2", 5, 12, "Be2"},
		{"queen d1-e2", 3, 12, "Qe2+"},
		{"king e1-e2", 4, 12, "Ke2"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := types.Move{From: tc.from, To: tc.to}
			san, err := Game.ToSan(m)
			if err != nil {
				t.Fatalf("ToSan error: %v", err)
			}
			if san != tc.want {
				t.Errorf("ToSan = %q, want %q", san, tc.want)
			}
		})
	}
}

func TestToSan_Castling(t *testing.T) {
	// White kingside castle: e1(4) -> g1(6)
	LoadFen("rnbqk2r/pppp1ppp/5n2/2b1p3/2B1P3/5N2/PPPP1PPP/RNBQK2R w KQkq - 4 4")
	m := types.Move{From: 4, To: 6, Flag: types.FlagCastleK}
	san, err := Game.ToSan(m)
	if err != nil {
		t.Fatalf("ToSan error: %v", err)
	}
	if san != "O-O" {
		t.Errorf("ToSan = %q, want %q", san, "O-O")
	}

	// White queenside castle: e1(4) -> c1(2)
	LoadFen("r3kbnr/pppqpppp/2n5/3p4/3P4/2N5/PPPQPPPP/R3KBNR w KQkq - 4 4")
	m = types.Move{From: 4, To: 2, Flag: types.FlagCastleQ}
	san, err = Game.ToSan(m)
	if err != nil {
		t.Fatalf("ToSan error: %v", err)
	}
	if san != "O-O-O" {
		t.Errorf("ToSan = %q, want %q", san, "O-O-O")
	}
}

func TestToSan_EnPassantCapture(t *testing.T) {
	// White pawn e5 captures d5 pawn en passant -> lands on d6
	// e5 = 36, d6 = 43, d5 pawn = 35
	LoadFen("rnbqkbnr/ppp1pppp/8/3pP3/8/8/PPPP1PPP/RNBQKBNR w KQkq d6 0 3")
	m := types.Move{From: 36, To: 43, Flag: types.FlagEnPassant, Captured: Game.Board[35]}
	san, err := Game.ToSan(m)
	if err != nil {
		t.Fatalf("ToSan error: %v", err)
	}
	if san != "exd6" {
		t.Errorf("ToSan = %q, want %q", san, "exd6")
	}
}

func TestToSan_Promotion(t *testing.T) {
	// White pawn on a7(48) promotes to a8(56)
	LoadFen("4k3/P7/8/8/8/8/8/4K3 w - - 0 1")
	m := types.Move{From: 48, To: 56, Flag: types.FlagPromotion, Promotion: types.Queen | types.ColorWhite}
	san, err := Game.ToSan(m)
	if err != nil {
		t.Fatalf("ToSan error: %v", err)
	}
	if san != "a8=Q+" {
		t.Errorf("ToSan = %q, want %q", san, "a8=Q+")
	}
}

func TestToSan_Disambiguation(t *testing.T) {
	// Two rooks on a1(0) and h1(7), both can reach d1(3).
	// King on e5 so it doesn't block the rook on h1.
	LoadFen("4k3/8/8/8/4K3/8/8/R6R w - - 0 1")
	m := types.Move{From: 0, To: 3}
	san, err := Game.ToSan(m)
	if err != nil {
		t.Fatalf("ToSan error: %v", err)
	}
	if san != "Rad1" {
		t.Errorf("ToSan = %q, want %q", san, "Rad1")
	}

	// Two white rooks on same file: a1(0) and a5(32), both go to a3(16)
	// a1 -> a3 needs rank disambiguation: "R1a3"
	LoadFen("4k3/8/8/R7/8/8/8/R3K3 w - - 0 1")
	m = types.Move{From: 0, To: 16}
	san, err = Game.ToSan(m)
	if err != nil {
		t.Fatalf("ToSan error: %v", err)
	}
	if san != "R1a3" {
		t.Errorf("ToSan = %q, want %q", san, "R1a3")
	}
}

func TestToSan_CheckSuffix(t *testing.T) {
	// Qd4(27) -> d8(59) gives check to king on e8(60)
	LoadFen("4k3/8/8/8/3Q4/8/8/4K3 w - - 0 1")
	m := types.Move{From: 27, To: 59}
	san, err := Game.ToSan(m)
	if err != nil {
		t.Fatalf("ToSan error: %v", err)
	}
	if san != "Qd8+" {
		t.Errorf("ToSan = %q, want %q", san, "Qd8+")
	}
}

func TestToSan_Checkmate(t *testing.T) {
	// Scholar's mate: Qh5(39) captures f7(53) = checkmate
	LoadFen("r1bqkb1r/pppp1ppp/2n2n2/4p2Q/2B1P3/8/PPPP1PPP/RNB1K1NR w KQkq - 4 4")
	m := types.Move{From: 39, To: 53}
	san, err := Game.ToSan(m)
	if err != nil {
		t.Fatalf("ToSan error: %v", err)
	}
	if san != "Qxf7#" {
		t.Errorf("ToSan = %q, want %q", san, "Qxf7#")
	}
}

func TestSanToMove_RoundTrip(t *testing.T) {
	LoadFen(StartingFEN)
	var ml MoveList
	Game.LegalMoves(&ml)

	for i := 0; i < ml.n; i++ {
		m := ml.moves[i]
		san, err := Game.ToSan(m)
		if err != nil {
			t.Errorf("ToSan error for move %v: %v", m, err)
			continue
		}
		matched, err := Game.SanToMove(san)
		if err != nil {
			t.Errorf("SanToMove error for SAN %q: %v", san, err)
			continue
		}
		if matched.From != m.From || matched.To != m.To {
			t.Errorf("SanToMove(%q) = {From:%d, To:%d}, want {From:%d, To:%d}", san, matched.From, matched.To, m.From, m.To)
		}
	}
}

func TestSanToMove_CastlingNotation(t *testing.T) {
	LoadFen("rnbqk2r/pppp1ppp/5n2/2b1p3/2B1P3/5N2/PPPP1PPP/RNBQK2R w KQkq - 4 4")
	m, err := Game.SanToMove("O-O")
	if err != nil {
		t.Fatalf("SanToMove error: %v", err)
	}
	if m.From != 4 || m.To != 6 {
		t.Errorf("SanToMove(O-O) = {From:%d, To:%d}, want {From:4, To:6}", m.From, m.To)
	}
}

func TestSanToMove_InvalidSAN(t *testing.T) {
	LoadFen(StartingFEN)
	_, err := Game.SanToMove("O-O")
	if err == nil {
		t.Error("SanToMove(O-O) should fail in starting position (no castling available)")
	}
}