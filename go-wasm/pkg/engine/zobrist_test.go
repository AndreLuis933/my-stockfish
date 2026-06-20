package engine

import (
	"testing"
	"webassemble/pkg/types"
)

func TestHashIncremental(t *testing.T) {
	positions := []string{
		StartingFEN,
		"rnbqkbnr/pppp1ppp/8/4p3/4P3/8/PPPP1PPP/RNBQKBNR w KQkq e6 0 2",
		"r1bqkbnr/pppp1ppp/2n5/4p3/2B1P3/5N2/PPPP1PPP/RNBQK2R w KQkq - 4 4",
		"r3k2r/p1ppqpb1/bn2Qnp1/3PN3/1p2P3/2N5/1PPPBBPP/R3K2R b KQkq - 0 1",
		"8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1",
	}

	for _, fen := range positions {
		var p Position
		p.LoadFen(fen)

		var ml MoveList
		p.LegalMoves(&ml)

		for i := 0; i < ml.Len(); i++ {
			m := ml.Get(i)
			p.Make(m)
			fullHash := p.ComputeHash()
			if p.Hash != fullHash {
				t.Errorf("hash mismatch after Make in %s move %d→%d: incremental=%d full=%d",
					fen, m.From, m.To, p.Hash, fullHash)
			}
			p.Unmake(m)
			if p.Hash != p.ComputeHash() {
				t.Errorf("hash mismatch after Unmake in %s: incremental=%d full=%d",
					fen, p.Hash, p.ComputeHash())
			}
		}
	}
}

func TestHashUnique(t *testing.T) {
	var p Position
	p.LoadFen(StartingFEN)

	var ml MoveList
	p.LegalMoves(&ml)

	hashes := make(map[uint64]bool)
	for i := 0; i < ml.Len(); i++ {
		m := ml.Get(i)
		p.Make(m)
		if hashes[p.Hash] {
			t.Errorf("duplicate hash for move %d→%d", m.From, m.To)
		}
		hashes[p.Hash] = true
		p.Unmake(m)
	}
}

func TestHashSideToMove(t *testing.T) {
	var p Position
	p.LoadFen(StartingFEN)
	h1 := p.Hash

	p.WhiteToMove = !p.WhiteToMove
	p.Hash = p.ComputeHash()
	h2 := p.Hash

	if h1 == h2 {
		t.Error("hash should differ when side to move changes")
	}
}

func TestHashPromotion(t *testing.T) {
	var p Position
	p.LoadFen("8/P7/8/8/8/8/8/4k2K w - - 0 1")

	var ml MoveList
	p.LegalMoves(&ml)
	found := false
	for i := 0; i < ml.Len(); i++ {
		m := ml.Get(i)
		if m.Flag == types.FlagPromotion {
			found = true
			p.Make(m)
			if p.Hash != p.ComputeHash() {
				t.Errorf("hash mismatch on promotion %d→%d promo=%d: inc=%d full=%d",
					m.From, m.To, m.Promotion, p.Hash, p.ComputeHash())
			}
			p.Unmake(m)
			if p.Hash != p.ComputeHash() {
				t.Errorf("hash mismatch after unmake promotion: inc=%d full=%d",
					p.Hash, p.ComputeHash())
			}
		}
	}
	if !found {
		t.Skip("no promotion moves found")
	}
}