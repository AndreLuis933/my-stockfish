package engine

import (
	"math/rand/v2"
)

// ---------------------------------------------------------------------------
// Magic bitboards for sliding pieces (rook and bishop).
//
// A sliding piece's attacks depend on board occupancy — a rook on d4 sees
// down the d-file until the first blocker. Precomputing every possible
// occupancy is infeasible naively (up to 2^12 per square). Magic bitboards
// solve this: a "magic" 64-bit multiplier hashes the relevant occupancy
// bits into a small, collision-free table index. The lookup is:
//
//   index = ((occupied & mask[sq]) * magic[sq]) >> shift[sq]
//   attacks = attackTable[offset[sq] + index]
//
// This file builds the masks, finds magics, and fills the attack tables
// in init() at program startup (~50-100ms one-time cost).
//
// Reference: https://www.chessprogramming.org/Magic_Bitboards
// ---------------------------------------------------------------------------

var (
	rookMask   [64]Bitboard
	bishopMask [64]Bitboard

	rookMagic   [64]uint64
	bishopMagic [64]uint64

	rookShift   [64]uint8
	bishopShift [64]uint8

	rookAttacksTable   []Bitboard
	bishopAttacksTable []Bitboard

	rookOffsets   [64]int
	bishopOffsets [64]int
)

// genSliderMask builds the relevant-occupancy mask for a slider on sq: the
// squares along each ray that could possibly be blockers, EXCLUDING the
// terminal square of each ray (the board edge). The terminal square is always
// part of the attack set regardless of occupancy (the ray always reaches it),
// so its occupancy doesn't affect the attack set and it can be excluded from
// the mask.
//
// For a rook on d4: the 4 rays each exclude their last square (d8, d1, a4, h4).
// For a rook on a1: the N ray excludes a8, the E ray excludes h1; S and W rays
// have no squares (a1 is already on the south/west edge).
func genSliderMask(sq int, directions [][2]int) Bitboard {
	var bb Bitboard
	rank, file := sq/8, sq%8
	for _, dir := range directions {
		dr, df := dir[0], dir[1]
		r, f := rank+dr, file+df
		// Walk the ray and add all squares EXCEPT the last one (the edge).
		// We detect "last one" by checking if stepping one more would go
		// off the board.
		for r >= 0 && r < 8 && f >= 0 && f < 8 {
			nr, nf := r+dr, f+df
			if nr < 0 || nr >= 8 || nf < 0 || nf >= 8 {
				break // this is the terminal square — don't add it
			}
			bb = setBit(bb, r*8+f)
			r = nr
			f = nf
		}
	}
	return bb
}

// rayAttacks computes the attack set for a slider on sq along one ray direction,
// given the full board occupancy. It walks the ray from sq until it hits a
// blocker (occupied square) or the board edge. The blocker square is included
// in the result (the slider attacks/captures it).
func rayAttacks(sq int, occ Bitboard, dr, df int) Bitboard {
	var bb Bitboard
	rank, file := sq/8, sq%8
	r, f := rank+dr, file+df
	for r >= 0 && r < 8 && f >= 0 && f < 8 {
		s := r*8 + f
		bb = setBit(bb, s)
		if isSet(occ, s) {
			break // blocker: include it, then stop
		}
		r += dr
		f += df
	}
	return bb
}

// sliderAttacksClassical computes the full attack set for a slider on sq
// using the classical ray-walk method. This is the REFERENCE implementation
// used to populate the magic attack table and to verify magic lookups in tests.
func sliderAttacksClassical(sq int, occ Bitboard, directions [][2]int) Bitboard {
	var bb Bitboard
	for _, dir := range directions {
		bb |= rayAttacks(sq, occ, dir[0], dir[1])
	}
	return bb
}

var rookDirs = [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}      // N, S, E, W
var bishopDirs = [][2]int{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}} // NE, NW, SE, SW

// ---------------------------------------------------------------------------
// Magic number search.
//
// For each square, we enumerate all 2^bits relevant-occupancy combinations.
// For each combination, we compute the reference attack set. Then we search
// for a 64-bit magic multiplier such that:
//
//   index = ((occ & mask) * magic) >> shift
//
// produces a unique index for every distinct attack set. If two different
// occupancies produce the same attack set, sharing an index is fine (no
// collision). We only fail if two different attack sets get the same index.
// ---------------------------------------------------------------------------

// enumerateMaskBits returns all set-bit indices of a bitboard. Used to iterate
// over the mask squares for occupancy enumeration.
func maskBits(mask Bitboard) []int {
	var sqs []int
	bb := mask
	for bb != 0 {
		sq := bitscan(bb)
		sqs = append(sqs, sq)
		bb &= bb - 1
	}
	return sqs
}

// findMagic searches for a magic multiplier that produces no index collisions
// for the given square's mask. Returns the magic, the shift, and the filled
// attack table for this square. The returned table slice has length 2^bits.
func findMagic(sq int, mask Bitboard, directions [][2]int) (uint64, uint8, []Bitboard) {
	bitsN := popcount(mask)
	shift := uint8(64 - bitsN)
	tableSize := 1 << bitsN

	maskSqs := maskBits(mask)
	numCombos := 1 << bitsN

	// Precompute reference attack sets for every relevant-occupancy combo.
	attacks := make([]Bitboard, numCombos)
	occs := make([]Bitboard, numCombos)
	for i := range numCombos {
		var occ Bitboard
		for bitIdx, sq2 := range maskSqs {
			if i&(1<<bitIdx) != 0 {
				occ = setBit(occ, sq2)
			}
		}
		occs[i] = occ
		attacks[i] = sliderAttacksClassical(sq, occ, directions)
	}

	// Search for a magic multiplier. Try random 64-bit values until we find
	// one that produces no bad collisions (two different attack sets mapping
	// to the same index). This typically takes a few hundred to a few
	// thousand tries.
	table := make([]Bitboard, tableSize)
	used := make([]bool, tableSize)
	// Use the last occupancy combo (all mask bits set) for the heuristic check.
	fullOcc := occs[numCombos-1]
	// Heuristic threshold: the product should spread at least half the mask
	// bits into the index range. For small masks (< 6 bits), use the mask
	// size itself to avoid an impossible threshold.
	threshold := bitsN / 2
	if threshold < 3 {
		threshold = bitsN
	}
	for attempts := 0; attempts < 100_000_000; attempts++ {
		magic := rand.Uint64() & rand.Uint64()
		if popcount(Bitboard((uint64(fullOcc&mask)*magic)>>shift)) < threshold {
			continue
		}

		for i := range tableSize {
			table[i] = 0
			used[i] = false
		}

		ok := true
		for i := range numCombos {
			idx := int((uint64(occs[i]&mask) * magic) >> shift)
			if idx >= tableSize {
				ok = false
				break
			}
			if used[idx] && table[idx] != attacks[i] {
				ok = false
				break
			}
			used[idx] = true
			table[idx] = attacks[i]
		}

		if ok {
			return magic, shift, table
		}
	}
	panic("findMagic: no magic found for square")
}

// ---------------------------------------------------------------------------
// Public attack functions. These are the runtime entry points used by the
// move generators once magic tables are built.
//
//   attacks = attackTable[offset[sq] + ((occ & mask[sq]) * magic[sq]) >> shift[sq]]
// ---------------------------------------------------------------------------

func rookAttacksBB(sq int, occ Bitboard) Bitboard {
	idx := int((uint64(occ&rookMask[sq]) * rookMagic[sq]) >> rookShift[sq])
	return rookAttacksTable[rookOffsets[sq]+idx]
}

func bishopAttacksBB(sq int, occ Bitboard) Bitboard {
	idx := int((uint64(occ&bishopMask[sq]) * bishopMagic[sq]) >> bishopShift[sq])
	return bishopAttacksTable[bishopOffsets[sq]+idx]
}

// init builds all magic tables at program startup.
func init() {
	// If hardcoded data is available (magic_data.go), use it.
	// Otherwise, fall back to runtime magic search.
	if useHardcodedMagic() {
		return
	}
	searchMagicAtRuntime()
}

// useHardcodedMagic loads pre-computed magic data from magic_data.go.
// Returns true if the data was loaded successfully.
func useHardcodedMagic() bool {
	if len(rookOffsetTable) == 0 {
		return false
	}
	rookMask = rookMaskData
	rookMagic = rookMagicData
	rookShift = rookShiftData
	rookOffsets = rookOffsetIndex
	rookAttacksTable = rookOffsetTable

	bishopMask = bishopMaskData
	bishopMagic = bishopMagicData
	bishopShift = bishopShiftData
	bishopOffsets = bishopOffsetIndex
	bishopAttacksTable = bishopOffsetTable
	return true
}

// searchMagicAtRuntime is the fallback magic search (used when magic_data.go
// is not present or empty). Takes ~16 seconds.
func searchMagicAtRuntime() {
	for sq := range 64 {
		rookMask[sq] = genSliderMask(sq, rookDirs)
		bishopMask[sq] = genSliderMask(sq, bishopDirs)
	}
	for sq := range 64 {
		magic, shift, table := findMagic(sq, rookMask[sq], rookDirs)
		rookMagic[sq] = magic
		rookShift[sq] = shift
		rookOffsets[sq] = len(rookAttacksTable)
		rookAttacksTable = append(rookAttacksTable, table...)

		magic, shift, table = findMagic(sq, bishopMask[sq], bishopDirs)
		bishopMagic[sq] = magic
		bishopShift[sq] = shift
		bishopOffsets[sq] = len(bishopAttacksTable)
		bishopAttacksTable = append(bishopAttacksTable, table...)
	}
}

// Dump helpers for the code generator (cmd/gen-magic).
func RookMaskForDump(sq int) Bitboard          { return rookMask[sq] }
func RookMagicForDump(sq int) uint64            { return rookMagic[sq] }
func RookShiftForDump(sq int) uint8             { return rookShift[sq] }
func RookOffsetForDump(sq int) int              { return rookOffsets[sq] }
func RookAttacksTableForDump() []Bitboard       { return rookAttacksTable }
func BishopMaskForDump(sq int) Bitboard         { return bishopMask[sq] }
func BishopMagicForDump(sq int) uint64          { return bishopMagic[sq] }
func BishopShiftForDump(sq int) uint8           { return bishopShift[sq] }
func BishopOffsetForDump(sq int) int            { return bishopOffsets[sq] }
func BishopAttacksTableForDump() []Bitboard     { return bishopAttacksTable }