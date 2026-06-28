package engine

import "testing"

func TestRookAttacksEmptyBoard(t *testing.T) {
	// With no blockers, a rook sees the full cross from its square.
	for sq := range 64 {
		got := rookAttacksBB(sq, 0)
		want := sliderAttacksClassical(sq, 0, rookDirs)
		if got != want {
			t.Errorf("rookAttacksBB(sq=%d, empty): got %016x, want %016x", sq, uint64(got), uint64(want))
		}
	}
}

func TestBishopAttacksEmptyBoard(t *testing.T) {
	for sq := range 64 {
		got := bishopAttacksBB(sq, 0)
		want := sliderAttacksClassical(sq, 0, bishopDirs)
		if got != want {
			t.Errorf("bishopAttacksBB(sq=%d, empty): got %016x, want %016x", sq, uint64(got), uint64(want))
		}
	}
}

func TestRookAttacksRandomOccupancies(t *testing.T) {
	// Test each square with several hand-crafted occupancies that exercise
	// blockers on every ray direction.
	testCases := []struct {
		sq  int
		occ Bitboard
	}{
		{0, 0},                                         // a1, empty
		{0, 1<<8 | 1<<1},                               // a1 blocked by a2 and b1
		{0, 1<<56},                                     // a1 blocked by a8 (far end)
		{0, 1<<7},                                      // a1 blocked by h1 (far end)
		{27, 0},                                        // d4, empty (center)
		{27, 1<<19 | 1<<28 | 1<<35 | 1<<3},             // d4 blocked on all 4 rays
		{27, 1<<35},                                    // d4 blocked only on north ray
		{28, 1<<20 | 1<<36},                            // e4 blocked on rank and file
		{63, 0},                                        // h8, empty
		{63, 1<<55 | 1<<62},                            // h8 blocked by h7 and g8
		{36, 0},                                        // e5, empty
		{36, 1<<44 | 1<<28 | 1<<37 | 1<<35},            // e5 blocked on all 4 rays
		{7, 0},                                         // h1, empty
		{7, 1<<15 | 1<<6},                              // h1 blocked by h2 and g1
		{56, 0},                                        // a8, empty
		{56, 1<<48 | 1<<57},                            // a8 blocked by a7 and b8
	}

	for _, tc := range testCases {
		got := rookAttacksBB(tc.sq, tc.occ)
		want := sliderAttacksClassical(tc.sq, tc.occ, rookDirs)
		if got != want {
			t.Errorf("rookAttacksBB(sq=%d, occ=%016x): got %016x, want %016x",
				tc.sq, uint64(tc.occ), uint64(got), uint64(want))
		}
	}
}

func TestBishopAttacksRandomOccupancies(t *testing.T) {
	testCases := []struct {
		sq  int
		occ Bitboard
	}{
		{0, 0},                    // a1, empty
		{0, 1<<9},                 // a1 blocked by b2
		{0, 1<<63},                // a1, h8 on diagonal but irrelevant
		{27, 0},                   // d4, empty (center)
		{27, 1<<18 | 1<<36 | 1<<20 | 1<<34}, // d4 blocked on all 4 diagonals
		{28, 0},                   // e4, empty
		{28, 1<<35},               // e4 blocked NE at e5... actually f5=37
		{28, 1<<37},               // e4 blocked NE at f5
		{63, 0},                   // h8, empty
		{63, 1<<54},               // h8 blocked by g7
		{36, 0},                   // e5, empty
		{36, 1<<27 | 1<<45 | 1<<29 | 1<<43}, // e5 blocked on all 4 diagonals
		{7, 0},                    // h1, empty
		{7, 1<<14},                // h1 blocked by g2
		{56, 0},                   // a8, empty
		{56, 1<<49},               // a8 blocked by b7
	}

	for _, tc := range testCases {
		got := bishopAttacksBB(tc.sq, tc.occ)
		want := sliderAttacksClassical(tc.sq, tc.occ, bishopDirs)
		if got != want {
			t.Errorf("bishopAttacksBB(sq=%d, occ=%016x): got %016x, want %016x",
				tc.sq, uint64(tc.occ), uint64(got), uint64(want))
		}
	}
}

func TestRookAttacksFullyBlocked(t *testing.T) {
	// Rook on a1 with all 4 adjacent squares occupied: should only see those 4.
	occ := Bitboard(1<<8 | 1<<1) // a2 and b1 — rook on a1 (sq 0)
	got := rookAttacksBB(0, occ)
	want := sliderAttacksClassical(0, occ, rookDirs)
	if got != want {
		t.Errorf("rookAttacksBB fully blocked: got %016x, want %016x", uint64(got), uint64(want))
	}
	// Verify it only sees 2 squares (a2 and b1, the two reachable blockers).
	if popcount(got) != 2 {
		t.Errorf("rookAttacksBB fully blocked: popcount = %d, want 2", popcount(got))
	}
}

func TestMagicTablesAllSquaresExhaustive(t *testing.T) {
	// For every square, test with all mask bits set (worst case: every
	// relevant blocker is present). The magic lookup must match classical.
	for sq := range 64 {
		fullOcc := rookMask[sq]
		got := rookAttacksBB(sq, fullOcc)
		want := sliderAttacksClassical(sq, fullOcc, rookDirs)
		if got != want {
			t.Errorf("rook sq=%d full block: got %016x, want %016x", sq, uint64(got), uint64(want))
		}

		fullOcc = bishopMask[sq]
		got = bishopAttacksBB(sq, fullOcc)
		want = sliderAttacksClassical(sq, fullOcc, bishopDirs)
		if got != want {
			t.Errorf("bishop sq=%d full block: got %016x, want %016x", sq, uint64(got), uint64(want))
		}
	}
}

func TestMagicTablesStartingPosition(t *testing.T) {
	// Occupancy of the starting position: all 32 pieces.
	startOcc := Bitboard(0xFFFF00000000FFFF)
	for sq := range 64 {
		// Only test squares where a piece actually sits (skip empties —
		// they still work but the test is more meaningful with blockers).
		got := rookAttacksBB(sq, startOcc)
		want := sliderAttacksClassical(sq, startOcc, rookDirs)
		if got != want {
			t.Errorf("rook sq=%d startpos: got %016x, want %016x", sq, uint64(got), uint64(want))
		}
		got = bishopAttacksBB(sq, startOcc)
		want = sliderAttacksClassical(sq, startOcc, bishopDirs)
		if got != want {
			t.Errorf("bishop sq=%d startpos: got %016x, want %016x", sq, uint64(got), uint64(want))
		}
	}
}

func TestMagicTableSizes(t *testing.T) {
	// The total table size should be reasonable (< 2MB for rooks, < 200KB for bishops).
	rookKB := len(rookAttacksTable) * 8 / 1024
	bishopKB := len(bishopAttacksTable) * 8 / 1024
	t.Logf("rook table: %d entries, %d KB", len(rookAttacksTable), rookKB)
	t.Logf("bishop table: %d entries, %d KB", len(bishopAttacksTable), bishopKB)
	if rookKB > 2048 {
		t.Errorf("rook table too large: %d KB", rookKB)
	}
	if bishopKB > 256 {
		t.Errorf("bishop table too large: %d KB", bishopKB)
	}
}