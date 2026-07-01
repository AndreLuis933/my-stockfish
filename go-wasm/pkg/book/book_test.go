package book

import (
	"math/rand"
	"testing"

	"webassemble/pkg/engine"
	"webassemble/pkg/types"
)

func TestLoadEmpty(t *testing.T) {
	b, err := Load(nil)
	if err != nil {
		t.Fatalf("Load(nil) error: %v", err)
	}
	if b.Len() != 0 {
		t.Fatalf("Len = %d, want 0", b.Len())
	}
}

func TestLoadInvalidSize(t *testing.T) {
	_, err := Load([]byte{0, 1, 2}) // 3 bytes, not multiple of 16
	if err == nil {
		t.Fatal("expected error for invalid file size")
	}
}

func TestLoadAndProbe(t *testing.T) {
	// Build a minimal book with 2 entries for the same position
	key := uint64(0x123456789ABCDEF0)
	entries := []Entry{
		{Key: key, Move: 0x1234, Weight: 60, Learn: 0},
		{Key: key, Move: 0x5678, Weight: 40, Learn: 0},
		{Key: key - 1, Move: 0x1111, Weight: 10, Learn: 0}, // different key, should not match
	}

	// Serialize to bytes
	data := make([]byte, len(entries)*EntrySize)
	for i, e := range entries {
		off := i * EntrySize
		binaryBigEndianPutUint64(data[off:], e.Key)
		binaryBigEndianPutUint16(data[off+8:], e.Move)
		binaryBigEndianPutUint16(data[off+10:], e.Weight)
		binaryBigEndianPutUint32(data[off+12:], e.Learn)
	}

	b, err := Load(data)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if b.Len() != 3 {
		t.Fatalf("Len = %d, want 3", b.Len())
	}

	// Probe for the key with 2 entries
	matches := b.Probe(key)
	if len(matches) != 2 {
		t.Fatalf("Probe returned %d matches, want 2", len(matches))
	}

	// Probe for a missing key
	matches = b.Probe(0xFFFFFFFFFFFFFFFF)
	if len(matches) != 0 {
		t.Fatalf("Probe returned %d matches for missing key, want 0", len(matches))
	}
}

func TestPickMoveWeighted(t *testing.T) {
	key := uint64(42)
	entries := []Entry{
		{Key: key, Move: 100, Weight: 90},
		{Key: key, Move: 200, Weight: 10},
	}

	data := make([]byte, len(entries)*EntrySize)
	for i, e := range entries {
		off := i * EntrySize
		binaryBigEndianPutUint64(data[off:], e.Key)
		binaryBigEndianPutUint16(data[off+8:], e.Move)
		binaryBigEndianPutUint16(data[off+10:], e.Weight)
		binaryBigEndianPutUint32(data[off+12:], e.Learn)
	}

	b, _ := Load(data)

	// With weight 90 vs 10, move 100 should be picked ~90% of the time
	rng := rand.New(rand.NewSource(42))
	pick100 := 0
	pick200 := 0
	for i := 0; i < 10000; i++ {
		move, ok := b.PickMove(key, rng)
		if !ok {
			t.Fatal("PickMove returned false")
		}
		switch move {
		case 100:
			pick100++
		case 200:
			pick200++
		default:
			t.Fatalf("unexpected move %d", move)
		}
	}

	// Should be approximately 9000/1000 (allow 10% tolerance)
	if pick100 < 8000 || pick100 > 10000 {
		t.Errorf("pick100 = %d, expected ~9000", pick100)
	}
	if pick200 < 500 || pick200 > 2000 {
		t.Errorf("pick200 = %d, expected ~1000", pick200)
	}
}

func TestPickMoveMiss(t *testing.T) {
	b, _ := Load(nil)
	_, ok := b.PickMove(12345, rand.New(rand.NewSource(1)))
	if ok {
		t.Fatal("PickMove on empty book should return false")
	}
}

func TestPickBestMove(t *testing.T) {
	key := uint64(99)
	entries := []Entry{
		{Key: key, Move: 100, Weight: 30},
		{Key: key, Move: 200, Weight: 70},
		{Key: key, Move: 300, Weight: 10},
	}

	data := make([]byte, len(entries)*EntrySize)
	for i, e := range entries {
		off := i * EntrySize
		binaryBigEndianPutUint64(data[off:], e.Key)
		binaryBigEndianPutUint16(data[off+8:], e.Move)
		binaryBigEndianPutUint16(data[off+10:], e.Weight)
		binaryBigEndianPutUint32(data[off+12:], e.Learn)
	}

	b, _ := Load(data)
	move, ok := b.PickBestMove(key)
	if !ok {
		t.Fatal("PickBestMove returned false")
	}
	if move != 200 {
		t.Fatalf("PickBestMove = %d, want 200 (highest weight)", move)
	}
}

func TestPolyglotHashStartPos(t *testing.T) {
	// Verify the polyglot hash of the start position matches python-chess
	engine.LoadFen(engine.StartingFEN)
	hash := PolyglotHash(engine.Game)

	// Known value: python -c "import chess; print(chess.polyglot.zobrist_hash(chess.Board()))"
	// = 0x463B96181691FC9C
	expected := uint64(0x463B96181691FC9C)
	if hash != expected {
		t.Fatalf("Start position hash = 0x%016X, want 0x%016X", hash, expected)
	}
}

func TestPolyglotHashAfterE4(t *testing.T) {
	// Load start position, push e2e4, verify hash
	engine.LoadFen(engine.StartingFEN)

	var ml engine.MoveList
	engine.Game.LegalMoves(&ml)

	var e2e4 types.Move
	found := false
	for i := 0; i < ml.Len(); i++ {
		m := ml.Get(i)
		if m.From == 12 && m.To == 28 { // e2=12, e4=28
			e2e4 = m
			found = true
			break
		}
	}
	if !found {
		t.Fatal("e2e4 not found in legal moves")
	}

	engine.Game.Make(e2e4)
	hash := PolyglotHash(engine.Game)

	// Known value: python -c "import chess; b=chess.Board(); b.push_san('e4'); print(hex(chess.polyglot.zobrist_hash(b)))"
	// = 0x823C9B50FD114196
	// Note: after 1.e4, ep target is e3 (sq=20) but no black pawn can capture it,
	// so the ep key is NOT included (polyglot convention).
	expected := uint64(0x823C9B50FD114196)
	if hash != expected {
		t.Fatalf("After e4 hash = 0x%016X, want 0x%016X", hash, expected)
	}
}

func TestDecodePolyglotMove(t *testing.T) {
	// e2e4: from=12, to=28, no promotion
	// encoded: to | (from << 6) | (0 << 12) = 28 | (12 << 6) = 28 + 768 = 796
	polyMove := uint16(28 | (12 << 6))
	move := DecodePolyglotMove(polyMove)
	if move.From != 12 {
		t.Errorf("From = %d, want 12", move.From)
	}
	if move.To != 28 {
		t.Errorf("To = %d, want 28", move.To)
	}
	if move.Promotion != 0 {
		t.Errorf("Promotion = %d, want 0", move.Promotion)
	}
}

func TestDecodePolyglotMovePromotion(t *testing.T) {
	// a7a8=Q: from=48, to=56, promo=4 (Queen)
	// encoded: 56 | (48 << 6) | (4 << 12) = 56 + 3072 + 16384 = 19512
	polyMove := uint16(56 | (48 << 6) | (4 << 12))
	move := DecodePolyglotMove(polyMove)
	if move.From != 48 {
		t.Errorf("From = %d, want 48", move.From)
	}
	if move.To != 56 {
		t.Errorf("To = %d, want 56", move.To)
	}
	if move.Promotion.TypePiece() != types.Queen {
		t.Errorf("Promotion type = %d, want Queen (%d)", move.Promotion.TypePiece(), types.Queen)
	}
}

func TestMatchLegalMove(t *testing.T) {
	engine.LoadFen(engine.StartingFEN)

	// e2e4 as a book move (from=12, to=28)
	bookMove := types.Move{From: 12, To: 28}
	matched, ok := MatchLegalMove(engine.Game, bookMove)
	if !ok {
		t.Fatal("MatchLegalMove failed for e2e4")
	}
	if matched.From != 12 || matched.To != 28 {
		t.Fatalf("matched = (from=%d, to=%d), want (12, 28)", matched.From, matched.To)
	}
}

func TestMatchLegalMoveMiss(t *testing.T) {
	engine.LoadFen(engine.StartingFEN)

	// Illegal move: a1 to h8 (a rook can't move there from a1 at start)
	bookMove := types.Move{From: 0, To: 63}
	_, ok := MatchLegalMove(engine.Game, bookMove)
	if ok {
		t.Fatal("MatchLegalMove should return false for illegal move")
	}
}

func TestLoadRealBook(t *testing.T) {
	// Try to load the actual book if it exists
	b, err := LoadFile("../../books/book.bin")
	if err != nil {
		t.Skipf("book.bin not available: %v", err)
	}

	if b.Len() == 0 {
		t.Fatal("book has 0 entries")
	}

	// The start position should have entries
	engine.LoadFen(engine.StartingFEN)
	hash := PolyglotHash(engine.Game)
	matches := b.Probe(hash)
	if len(matches) == 0 {
		t.Fatal("no book entries for start position")
	}

	t.Logf("Book: %d entries, start position has %d moves", b.Len(), len(matches))
	for _, e := range matches {
		move := DecodePolyglotMove(e.Move)
		t.Logf("  from=%d to=%d promo=%d weight=%d", move.From, move.To, move.Promotion, e.Weight)
	}
}

// Helpers for building test book data

func binaryBigEndianPutUint64(b []byte, v uint64) {
	b[0] = byte(v >> 56)
	b[1] = byte(v >> 48)
	b[2] = byte(v >> 40)
	b[3] = byte(v >> 32)
	b[4] = byte(v >> 24)
	b[5] = byte(v >> 16)
	b[6] = byte(v >> 8)
	b[7] = byte(v)
}

func binaryBigEndianPutUint16(b []byte, v uint16) {
	b[0] = byte(v >> 8)
	b[1] = byte(v)
}

func binaryBigEndianPutUint32(b []byte, v uint32) {
	b[0] = byte(v >> 24)
	b[1] = byte(v >> 16)
	b[2] = byte(v >> 8)
	b[3] = byte(v)
}