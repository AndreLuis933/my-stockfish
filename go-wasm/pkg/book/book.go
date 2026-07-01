package book

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"os"
	"sort"
)

// EntrySize is the size of a single Polyglot book entry in bytes (16).
const EntrySize = 16

// Entry is a single Polyglot book entry.
//
// Layout (16 bytes, big-endian):
//
//	key    uint64 (8) — Polyglot Zobrist hash of the position
//	move   uint16 (2) — encoded move (polyglot format)
//	weight uint16 (2) — relative weight (higher = more likely to be picked)
//	learn  uint32 (4) — reserved (usually 0)
type Entry struct {
	Key    uint64
	Move   uint16
	Weight uint16
	Learn  uint32
}

// Book is an in-memory Polyglot opening book.
// Entries are sorted by Key for binary-search probing.
type Book struct {
	entries []Entry
}

// Load parses a Polyglot .bin file from a byte slice.
// The file must be a multiple of EntrySize (16 bytes).
func Load(data []byte) (*Book, error) {
	if len(data)%EntrySize != 0 {
		return nil, fmt.Errorf("invalid book file size %d: not a multiple of %d", len(data), EntrySize)
	}

	count := len(data) / EntrySize
	entries := make([]Entry, count)

	for i := 0; i < count; i++ {
		off := i * EntrySize
		entries[i] = Entry{
			Key:    binary.BigEndian.Uint64(data[off:]),
			Move:   binary.BigEndian.Uint16(data[off+8:]),
			Weight: binary.BigEndian.Uint16(data[off+10:]),
			Learn:  binary.BigEndian.Uint32(data[off+12:]),
		}
	}

	// Entries should already be sorted by key in a valid .bin file,
	// but we sort to be safe (binary search requires it).
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Key < entries[j].Key
	})

	return &Book{entries: entries}, nil
}

// LoadFile loads a Polyglot .bin file from disk.
func LoadFile(path string) (*Book, error) {
	data, err := os.ReadFile(path) // #nosec G304 — path is user/engine config
	if err != nil {
		return nil, fmt.Errorf("read book file: %w", err)
	}
	return Load(data)
}

// Len returns the number of entries in the book.
func (b *Book) Len() int {
	return len(b.entries)
}

// Probe returns all entries matching the given hash.
// Uses binary search to find the first match, then collects consecutive entries.
func (b *Book) Probe(hash uint64) []Entry {
	// Binary search for the first entry with key >= hash
	idx := sort.Search(len(b.entries), func(i int) bool {
		return b.entries[i].Key >= hash
	})

	// Collect all consecutive entries with matching key
	var result []Entry
	for idx < len(b.entries) && b.entries[idx].Key == hash {
		result = append(result, b.entries[idx])
		idx++
	}

	return result
}

// PickMove returns a weighted-random book move for the given hash.
// Returns the polyglot-encoded move and true on hit, or 0 and false on miss.
//
// The selection is weighted: moves with higher Weight are more likely to be
// picked. This adds variety across games while favoring stronger/more popular
// moves.
func (b *Book) PickMove(hash uint64, rng *rand.Rand) (uint16, bool) {
	matches := b.Probe(hash)
	if len(matches) == 0 {
		return 0, false
	}

	// Sum all weights
	totalWeight := 0
	for _, e := range matches {
		totalWeight += int(e.Weight)
	}

	if totalWeight == 0 {
		// All weights are 0 — pick uniformly
		return matches[rng.Intn(len(matches))].Move, true
	}

	// Weighted random selection
	pick := rng.Intn(totalWeight)
	cumulative := 0
	for _, e := range matches {
		cumulative += int(e.Weight)
		if cumulative > pick {
			return e.Move, true
		}
	}

	// Fallback (shouldn't reach here)
	return matches[len(matches)-1].Move, true
}

// PickBestMove returns the highest-weighted book move for the given hash.
// Returns the polyglot-encoded move and true on hit, or 0 and false on miss.
func (b *Book) PickBestMove(hash uint64) (uint16, bool) {
	matches := b.Probe(hash)
	if len(matches) == 0 {
		return 0, false
	}

	best := matches[0]
	for _, e := range matches[1:] {
		if e.Weight > best.Weight {
			best = e
		}
	}

	return best.Move, true
}