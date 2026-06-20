package engine

import "webassemble/pkg/types"

// TTFlag indicates how the stored score should be interpreted:
//   - TTExact: the score is the exact value of the position
//   - TTLower: the score is a lower bound (a beta cutoff happened — the
//     position is at least this good for the side to move)
//   - TTUpper: the score is an upper bound (alpha didn't improve — the
//     position is at most this good for the side to move)
type TTFlag uint8

const (
	TTNone   TTFlag = 0
	TTExact  TTFlag = 1
	TTLower  TTFlag = 2 // beta bound: score >= beta, move caused cutoff
	TTUpper  TTFlag = 3 // alpha bound: score <= alpha, no move improved
)

// TTEntry is 16 bytes — fits in a single cache line (on 64-byte lines, 4 entries
// per line). The key is the full 64-bit Zobrist hash for collision verification.
// Score is stored as int16 (adjusted for mate distance at store time).
// Move is packed as from(6) | to(6) | promo(4) = 16 bits.
type TTEntry struct {
	Key   uint64
	Score int16
	Depth uint8
	Flag  TTFlag
	Move  uint16
}

// PackMove encodes a Move into a uint16 for TT storage.
func PackMove(m types.Move) uint16 {
	promo := 0
	if m.Promotion != 0 {
		switch m.Promotion & types.TypeMask {
		case types.Queen:
			promo = 1
		case types.Rook:
			promo = 2
		case types.Bishop:
			promo = 3
		case types.Knight:
			promo = 4
		}
	}
	return uint16(m.From)<<10 | uint16(m.To)<<4 | uint16(promo)
}

// UnpackMove decodes a uint16 from the TT back into a Move.
// Returns ok=false if the packed value is 0 (no move stored).
func UnpackMove(packed uint16) (types.Move, bool) {
	if packed == 0 {
		return types.Move{}, false
	}
	from := int(packed >> 10)
	to := int((packed >> 4) & 0x3F)
	promoCode := int(packed & 0xF)

	var promotion types.Piece
	switch promoCode {
	case 1:
		promotion = types.Queen
	case 2:
		promotion = types.Rook
	case 3:
		promotion = types.Bishop
	case 4:
		promotion = types.Knight
	}

	flag := types.FlagNormal
	if promoCode != 0 {
		flag = types.FlagPromotion
	}

	return types.Move{
		From:      from,
		To:        to,
		Promotion: promotion,
		Flag:      flag,
	}, true
}

// TranspositionTable is a fixed-size array of TTEntries, probed by
// hash & mask. Always-replace strategy: a new entry overwrites the old one
// at the same slot. This is simple and works well at our search depth.
type TranspositionTable struct {
	entries []TTEntry
	mask    uint64
	used    int // number of slots written at least once
}

// NewTranspositionTable creates a TT with approximately sizeBytes of memory.
// sizeBytes is rounded down to the nearest power-of-2 entry count.
func NewTranspositionTable(sizeBytes int) *TranspositionTable {
	entryCount := sizeBytes / 16
	// Round down to power of 2.
	power := 1
	for power*2 <= entryCount {
		power *= 2
	}
	if power < 1 {
		power = 1
	}
	return &TranspositionTable{
		entries: make([]TTEntry, power),
		mask:    uint64(power - 1),
	}
}

// DefaultTranspositionTable returns a 4MB TT (256K entries). Smaller than the
// theoretical ideal but allocates fast enough for short time controls.
func DefaultTranspositionTable() *TranspositionTable {
	return NewTranspositionTable(4 * 1024 * 1024)
}

// Probe returns the entry for the given hash, or ok=false if no entry
// exists or the key doesn't match (index collision with a different position).
func (tt *TranspositionTable) Probe(hash uint64) (TTEntry, bool) {
	entry := &tt.entries[hash&tt.mask]
	if entry.Key != hash || entry.Flag == TTNone {
		return TTEntry{}, false
	}
	return *entry, true
}

// Store writes an entry. Always-replace: overwrites whatever is in the slot.
// If the slot was empty (Flag==TTNone), increments the used counter.
func (tt *TranspositionTable) Store(hash uint64, depth uint8, score int16, move uint16, flag TTFlag) {
	idx := hash & tt.mask
	if tt.entries[idx].Flag == TTNone {
		tt.used++
	}
	tt.entries[idx] = TTEntry{
		Key:   hash,
		Score: score,
		Depth: depth,
		Flag:  flag,
		Move:  move,
	}
}

// Clear resets all entries. Called between games.
func (tt *TranspositionTable) Clear() {
	for i := range tt.entries {
		tt.entries[i] = TTEntry{}
	}
	tt.used = 0
}

// FillPercent returns the percentage of slots that have been written at least
// once. A high fill rate means more index collisions (replacements), which
// reduces TT effectiveness — a sign the table should be bigger.
func (tt *TranspositionTable) FillPercent() float64 {
	if len(tt.entries) == 0 {
		return 0
	}
	return float64(tt.used) / float64(len(tt.entries)) * 100
}

// Size returns the table capacity in entries.
func (tt *TranspositionTable) Size() int {
	return len(tt.entries)
}