package engine

import (
	"testing"

	"webassemble/pkg/types"
)

func benchmarkPerft(b *testing.B, depth int, fn func(int) int) {
	LoadFen(StartingFEN)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fn(depth)
	}
}

// BenchmarkPerftBulkDepth4 counts perft nodes per second using the bulk-counting
// method (no move list at the leaf — just count legal moves).
func BenchmarkPerftBulkDepth4(b *testing.B) {
	benchmarkPerft(b, 4, Game.Perft)
}

// BenchmarkPerftFastDepth4 counts perft nodes per second using the lightweight
// make/unmake path.
func BenchmarkPerftFastDepth4(b *testing.B) {
	benchmarkPerft(b, 4, Game.PerftFast)
}

// BenchmarkPerftBulkDepth5 measures deeper perft.
func BenchmarkPerftBulkDepth5(b *testing.B) {
	benchmarkPerft(b, 5, Game.Perft)
}

// BenchmarkPerftFastDepth5 measures deeper perft with the fast path.
func BenchmarkPerftFastDepth5(b *testing.B) {
	benchmarkPerft(b, 5, Game.PerftFast)
}

// BenchmarkPseudoLegalCaptures benchmarks quiescence capture gen.
func BenchmarkPseudoLegalCaptures(b *testing.B) {
	LoadFen("r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1")
	var ml MoveList
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Game.PseudoLegalCaptures(&ml)
	}
}

// BenchmarkLegalMovesKiwipete benchmarks legal moves from a complex position.
func BenchmarkLegalMovesKiwipete(b *testing.B) {
	LoadFen("r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1")
	var ml MoveList
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Game.LegalMoves(&ml)
	}
}

// BenchmarkMakeUnmakeCapture benchmarks make/unmake with a capture move.
func BenchmarkMakeUnmakeCapture(b *testing.B) {
	LoadFen("r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1")
	var ml MoveList
	Game.LegalMoves(&ml)
	var capture types.Move
	for i := 0; i < ml.Len(); i++ {
		m := ml.Get(i)
		if m.Captured != 0 {
			capture = m
			break
		}
	}
	if capture.From == 0 && capture.To == 0 {
		b.Fatal("no capture found")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Game.Make(capture)
		Game.Unmake(capture)
	}
}

// BenchmarkIsInCheck benchmarks check detection (called at every node).
func BenchmarkIsInCheck(b *testing.B) {
	LoadFen(StartingFEN)
	color := Game.colorOfSide()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Game.IsInCheck(color)
	}
}