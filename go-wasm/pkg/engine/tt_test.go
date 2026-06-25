package engine

import (
	"testing"
	"unsafe"
)

// TestTTEntrySize verifies that TTEntry stays at 16 bytes. The Gen field was
// added for gen-aware replacement — it must fit in the natural struct padding
// (between Flag and Move, which needs 2-byte alignment) without growing the
// struct. If this test fails, the TT is larger than expected, wasting cache
// lines and memory.
func TestTTEntrySize(t *testing.T) {
	if size := unsafe.Sizeof(TTEntry{}); size != TTEntrySize {
		t.Errorf("TTEntry size = %d bytes, want %d (Gen field must fit in padding)",
			size, TTEntrySize)
	}
}