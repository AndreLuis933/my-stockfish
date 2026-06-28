package engine

import (
	"fmt"
	"testing"
	"time"

	"webassemble/pkg/types"
)

// BenchmarkMoveGen benchmarks full pseudo-legal move generation.
func BenchmarkPseudoLegalMoves(b *testing.B) {
	LoadFen(StartingFEN)
	var ml MoveList
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Game.PseudoLegalMoves(&ml)
	}
}

// BenchmarkLegalMoves benchmarks full legal move generation (Make/Unmake filter).
func BenchmarkLegalMoves(b *testing.B) {
	LoadFen(StartingFEN)
	var ml MoveList
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Game.LegalMoves(&ml)
	}
}

// BenchmarkAttackersTo benchmarks check detection via bitboards.
func BenchmarkAttackersTo(b *testing.B) {
	LoadFen(StartingFEN)
	kingSq := Game.KingSquares[1] // black king
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Game.attackersTo(kingSq, types.ColorWhite)
	}
}

// BenchmarkIsSquareAttacked benchmarks the public API (same as attackersTo != 0).
func BenchmarkIsSquareAttacked(b *testing.B) {
	LoadFen(StartingFEN)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Game.IsSquareAttacked(28, types.ColorWhite)
	}
}

// BenchmarkRookAttacks benchmarks a single rook magic lookup.
func BenchmarkRookAttacks(b *testing.B) {
	occ := Bitboard(0xFFFF00000000FFFF) // starting position occupancy
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rookAttacksBB(0, occ)
	}
}

// BenchmarkBishopAttacks benchmarks a single bishop magic lookup.
func BenchmarkBishopAttacks(b *testing.B) {
	occ := Bitboard(0xFFFF00000000FFFF)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bishopAttacksBB(2, occ)
	}
}

// BenchmarkMakeUnmake benchmarks a single Make + Unmake cycle.
func BenchmarkMakeUnmake(b *testing.B) {
	LoadFen(StartingFEN)
	var ml MoveList
	Game.LegalMoves(&ml)
	move := ml.Get(0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Game.Make(move)
		Game.Unmake(move)
	}
}

// TestMagicInitTime measures how long the magic bitboard table generation
// takes at startup. Since init() already ran, we re-run findMagic for all
// squares and time it.
func TestMagicInitTime(t *testing.T) {
	// Re-run the full magic search for all 64 squares (rooks + bishops)
	// to measure startup cost.
	start := time.Now()
	for sq := range 64 {
		_, _, _ = findMagic(sq, rookMask[sq], rookDirs)
	}
	rookElapsed := time.Since(start)

	start = time.Now()
	for sq := range 64 {
		_, _, _ = findMagic(sq, bishopMask[sq], bishopDirs)
	}
	bishopElapsed := time.Since(start)

	t.Logf("magic init time: rooks=%v, bishops=%v, total=%v",
		rookElapsed, bishopElapsed, rookElapsed+bishopElapsed)
	t.Logf("rook table: %d entries (%d KB)", len(rookAttacksTable), len(rookAttacksTable)*8/1024)
	t.Logf("bishop table: %d entries (%d KB)", len(bishopAttacksTable), len(bishopAttacksTable)*8/1024)
}

// TestSliderAttackSpeed measures how many slider attack lookups per second.
func TestSliderAttackSpeed(t *testing.T) {
	LoadFen(StartingFEN)
	occ := Game.Occupied

	const iterations = 10_000_000

	// Rook attacks from every square.
	start := time.Now()
	for i := 0; i < iterations; i++ {
		sq := i & 63
		_ = rookAttacksBB(sq, occ)
	}
	rookElapsed := time.Since(start)
	rookNps := int64(iterations) * int64(time.Second) / int64(rookElapsed)

	// Bishop attacks from every square.
	start = time.Now()
	for i := 0; i < iterations; i++ {
		sq := i & 63
		_ = bishopAttacksBB(sq, occ)
	}
	bishopElapsed := time.Since(start)
	bishopNps := int64(iterations) * int64(time.Second) / int64(bishopElapsed)

	t.Logf("rook attacks:   %d lookups in %v (%d Mlookups/sec)", iterations, rookElapsed, rookNps/1_000_000)
	t.Logf("bishop attacks: %d lookups in %v (%d Mlookups/sec)", iterations, bishopElapsed, bishopNps/1_000_000)
}

// TestMoveGenSpeed measures moves generated per second.
func TestMoveGenSpeed(t *testing.T) {
	LoadFen(StartingFEN)
	var ml MoveList

	// Warm up.
	for i := 0; i < 1000; i++ {
		Game.PseudoLegalMoves(&ml)
	}

	const iterations = 1_000_000
	start := time.Now()
	totalMoves := 0
	for i := 0; i < iterations; i++ {
		Game.PseudoLegalMoves(&ml)
		totalMoves += ml.Len()
	}
	elapsed := time.Since(start)
	nps := int64(iterations) * int64(time.Second) / int64(elapsed)

	t.Logf("PseudoLegalMoves: %d calls in %v (%d calls/sec, avg %d moves/pos, %d Mmoves/sec)",
		iterations, elapsed, nps/1_000_000, totalMoves/iterations, int64(totalMoves)*int64(time.Second)/int64(elapsed)/1_000_000)
}

// TestAttackersToSpeed measures attackersTo calls per second.
func TestAttackersToSpeed(t *testing.T) {
	LoadFen(StartingFEN)

	// Test from multiple squares (king squares, center, etc).
	squares := []int{4, 60, 28, 36, 0, 63}
	colors := []types.Piece{types.ColorWhite, types.ColorBlack}

	const iterations = 1_000_000
	start := time.Now()
	for i := 0; i < iterations; i++ {
		sq := squares[i%len(squares)]
		col := colors[i%len(colors)]
		Game.attackersTo(sq, col)
	}
	elapsed := time.Since(start)
	nps := int64(iterations) * int64(time.Second) / int64(elapsed)

	t.Logf("attackersTo: %d calls in %v (%d Mcalls/sec)", iterations, elapsed, nps/1_000_000)
}

// TestMakeUnmakeSpeed measures Make/Unmake cycles per second.
func TestMakeUnmakeSpeed(t *testing.T) {
	LoadFen(StartingFEN)
	var ml MoveList
	Game.LegalMoves(&ml)

	if ml.Len() == 0 {
		t.Fatal("no moves")
	}
	move := ml.Get(0)

	const iterations = 10_000_000
	start := time.Now()
	for i := 0; i < iterations; i++ {
		Game.Make(move)
		Game.Unmake(move)
	}
	elapsed := time.Since(start)
	nps := int64(iterations) * int64(time.Second) / int64(elapsed)

	t.Logf("Make+Unmake: %d cycles in %v (%d Mcycles/sec)", iterations, elapsed, nps/1_000_000)
}

// TestTableSizes prints the magic table sizes.
func TestTableSizes(t *testing.T) {
	t.Logf("rook table:   %d entries, %d KB (%d bytes)", len(rookAttacksTable), len(rookAttacksTable)*8/1024, len(rookAttacksTable)*8)
	t.Logf("bishop table: %d entries, %d KB (%d bytes)", len(bishopAttacksTable), len(bishopAttacksTable)*8/1024, len(bishopAttacksTable)*8)
	t.Logf("total magic:  %d KB", (len(rookAttacksTable)+len(bishopAttacksTable))*8/1024)
	fmt.Printf("rook table:   %d entries, %d KB\n", len(rookAttacksTable), len(rookAttacksTable)*8/1024)
	fmt.Printf("bishop table: %d entries, %d KB\n", len(bishopAttacksTable), len(bishopAttacksTable)*8/1024)
	fmt.Printf("total magic:  %d KB\n", (len(rookAttacksTable)+len(bishopAttacksTable))*8/1024)
}