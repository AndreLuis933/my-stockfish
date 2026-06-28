package engine

import "math/bits"

// Bitboard is a set of squares: bit i set means "square i is in this set".
// Square index uses the LERF convention: a1=0, h1=7, a2=8, ..., h8=63.
type Bitboard uint64

// ---------------------------------------------------------------------------
// File and rank masks. Used to prevent wrap-around in shift-based move gen
// (e.g., a pawn on file H must not "capture" to file A of the next rank).
// ---------------------------------------------------------------------------

var (
	fileA Bitboard = 0x0101010101010101
	fileH Bitboard = 0x8080808080808080
	notA  Bitboard = ^fileA
	notH  Bitboard = ^fileH
)

func rankMask(r int) Bitboard { return Bitboard(0xFF) << (r * 8) }

func fileMask(f int) Bitboard { return fileA << f }

// ---------------------------------------------------------------------------
// Single-bit operations: the three atoms every bitboard computation builds on.
// ---------------------------------------------------------------------------

func setBit(bb Bitboard, sq int) Bitboard   { return bb | (1 << sq) }
func clearBit(bb Bitboard, sq int) Bitboard { return bb &^ (1 << sq) }
func isSet(bb Bitboard, sq int) bool        { return bb&(1<<sq) != 0 }

func popcount(bb Bitboard) int { return bits.OnesCount64(uint64(bb)) }

// bitscan returns the index of the lowest set bit. Combined with the
// "clear lowest bit" trick (bb & (bb - 1)), this is how you iterate over
// only the set bits of a bitboard without scanning all 64.
func bitscan(bb Bitboard) int { return bits.TrailingZeros64(uint64(bb)) }

// ---------------------------------------------------------------------------
// Precomputed attack tables for non-sliders. Built once at startup via init().
// knightAttacks[sq] = bitboard of every square a knight on sq can jump to.
// kingAttacks[sq]   = bitboard of every square a king on sq can step to.
// ---------------------------------------------------------------------------

var knightAttacks [64]Bitboard
var kingAttacks [64]Bitboard

// genStepper builds an attack bitboard for a piece that moves by fixed
// (delta-rank, delta-file) offsets. It checks board boundaries in (rank, file)
// space, which naturally prevents wrap-around across file edges.
func genStepper(sq int, offsets [][2]int) Bitboard {
	var bb Bitboard
	rank, file := sq/8, sq%8
	for _, off := range offsets {
		tr, tf := rank+off[0], file+off[1]
		if tr < 0 || tr > 7 || tf < 0 || tf > 7 {
			continue
		}
		bb = setBit(bb, tr*8+tf)
	}
	return bb
}

func init() {
	knightOffsets := [][2]int{{-2, -1}, {-2, 1}, {-1, -2}, {-1, 2}, {1, -2}, {1, 2}, {2, -1}, {2, 1}}
	kingOffsets := [][2]int{{-1, -1}, {-1, 0}, {-1, 1}, {0, -1}, {0, 1}, {1, -1}, {1, 0}, {1, 1}}
	for sq := range 64 {
		knightAttacks[sq] = genStepper(sq, knightOffsets)
		kingAttacks[sq] = genStepper(sq, kingOffsets)
	}
}