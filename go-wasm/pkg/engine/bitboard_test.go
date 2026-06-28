package engine

import "testing"

func TestKnightAttacks(t *testing.T) {
	tests := []struct {
		sq      int
		name    string
		wantSqs []int
	}{
		{6, "g1", []int{12, 21, 23}},        // e2, f3, h3
		{0, "a1", []int{10, 17}},            // c2, b3 — no wrap
		{63, "h8", []int{46, 53}},           // g6, f7 — no wrap
		{28, "e4", []int{11, 13, 18, 22, 34, 38, 43, 45}},
		{7, "h1", []int{13, 22}},            // f2, g3 — no wrap
		{56, "a8", []int{41, 50}},           // b6, c7 — no wrap
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := knightAttacks[tc.sq]
			for _, s := range tc.wantSqs {
				if !isSet(got, s) {
					t.Errorf("knightAttacks[%s] (sq %d): expected square %d to be set, but it wasn't", tc.name, tc.sq, s)
				}
			}
			if popcount(got) != len(tc.wantSqs) {
				t.Errorf("knightAttacks[%s]: popcount = %d, want %d", tc.name, popcount(got), len(tc.wantSqs))
			}
		})
	}
}

func TestKingAttacks(t *testing.T) {
	tests := []struct {
		sq      int
		name    string
		wantSqs []int
	}{
		{4, "e1", []int{3, 5, 11, 12, 13}},  // d1, f1, d2, e2, f2
		{0, "a1", []int{1, 8, 9}},           // b1, a2, b2 — corner
		{63, "h8", []int{54, 55, 62}},       // g7, h7, g8 — corner
		{28, "e4", []int{19, 20, 21, 27, 29, 35, 36, 37}},
		{7, "h1", []int{6, 14, 15}},         // g1, g2, h2 — edge
		{56, "a8", []int{48, 49, 57}},       // a7, b7, b8 — edge
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := kingAttacks[tc.sq]
			for _, s := range tc.wantSqs {
				if !isSet(got, s) {
					t.Errorf("kingAttacks[%s] (sq %d): expected square %d to be set, but it wasn't", tc.name, tc.sq, s)
				}
			}
			if popcount(got) != len(tc.wantSqs) {
				t.Errorf("kingAttacks[%s]: popcount = %d, want %d", tc.name, popcount(got), len(tc.wantSqs))
			}
		})
	}
}

func TestKnightKingNoOverlap(t *testing.T) {
	for sq := range 64 {
		if knightAttacks[sq]&kingAttacks[sq] != 0 {
			t.Errorf("knight and king attacks overlap on square %d", sq)
		}
	}
}

func TestKnightAttacksAllSquares(t *testing.T) {
	knightOffsets := [][2]int{{-2, -1}, {-2, 1}, {-1, -2}, {-1, 2}, {1, -2}, {1, 2}, {2, -1}, {2, 1}}
	for sq := range 64 {
		pc := popcount(knightAttacks[sq])
		rank, file := sq/8, sq%8
		want := 0
		for _, off := range knightOffsets {
			tr, tf := rank+off[0], file+off[1]
			if tr >= 0 && tr < 8 && tf >= 0 && tf < 8 {
				want++
			}
		}
		if pc != want {
			t.Errorf("knightAttacks[sq %d]: popcount = %d, want %d", sq, pc, want)
		}
	}
}

func TestKingAttacksAllSquares(t *testing.T) {
	for sq := range 64 {
		pc := popcount(kingAttacks[sq])
		rank, file := sq/8, sq%8
		var want int
		switch {
		case (rank == 0 || rank == 7) && (file == 0 || file == 7):
			want = 3 // corner
		case rank == 0 || rank == 7 || file == 0 || file == 7:
			want = 5 // edge
		default:
			want = 8 // center
		}
		if pc != want {
			t.Errorf("kingAttacks[sq %d]: popcount = %d, want %d", sq, pc, want)
		}
	}
}

func TestPopcount(t *testing.T) {
	if popcount(0) != 0 {
		t.Error("popcount(0) != 0")
	}
	if popcount(0xFFFFFFFFFFFFFFFF) != 64 {
		t.Error("popcount(full) != 64")
	}
	if popcount(1) != 1 {
		t.Error("popcount(1) != 1")
	}
}

func TestBitscan(t *testing.T) {
	if bitscan(0b1000) != 3 {
		t.Error("bitscan(0b1000) != 3")
	}
	if bitscan(0b1010) != 1 {
		t.Error("bitscan(0b1010) != 1")
	}
	if bitscan(1<<63) != 63 {
		t.Error("bitscan(1<<63) != 63")
	}
}

func TestSetClearIsSet(t *testing.T) {
	var bb Bitboard
	bb = setBit(bb, 28)
	if !isSet(bb, 28) {
		t.Error("setBit failed")
	}
	if isSet(bb, 27) {
		t.Error("isSet false positive")
	}
	bb = clearBit(bb, 28)
	if isSet(bb, 28) {
		t.Error("clearBit failed")
	}
}

func TestFileRankMasks(t *testing.T) {
	if popcount(fileA) != 8 {
		t.Error("fileA popcount != 8")
	}
	if popcount(fileH) != 8 {
		t.Error("fileH popcount != 8")
	}
	if popcount(rankMask(0)) != 8 {
		t.Error("rank1 popcount != 8")
	}
	if popcount(rankMask(7)) != 8 {
		t.Error("rank8 popcount != 8")
	}
	for sq := range 64 {
		rank, file := sq/8, sq%8
		if isSet(fileA, sq) != (file == 0) {
			t.Errorf("fileA mismatch at sq %d", sq)
		}
		if isSet(fileH, sq) != (file == 7) {
			t.Errorf("fileH mismatch at sq %d", sq)
		}
		if isSet(rankMask(rank), sq) != true {
			t.Errorf("rankMask(%d) missing sq %d", rank, sq)
		}
		if isSet(rankMask((rank+1)%8), sq) {
			t.Errorf("rankMask wrong rank at sq %d", sq)
		}
	}
}