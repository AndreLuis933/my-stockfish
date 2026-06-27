package main

import (
	"fmt"
	"math/bits"
)

// Bitboard is a set of squares: bit i set means "square i is in this set".
// Square index = rank*8 + file (LERF convention): a1=0, h1=7, a8=56, h8=63.
type Bitboard uint64

// ---------------------------------------------------------------------------
// File and rank masks.
// fileA has bits 0, 8, 16, 24, 32, 40, 48, 56 set (every a-square).
// fileH has bits 7, 15, 23, 31, 39, 47, 55, 63 set (every h-square).
// rankN has 8 consecutive bits set, shifted into place.
// ---------------------------------------------------------------------------

const (
	fileA Bitboard = 0x0101010101010101
	fileH Bitboard = 0x8080808080808080
	notA  Bitboard = ^fileA
	notH  Bitboard = ^fileH
)

func rankMask(r int) Bitboard { return Bitboard(0xFF) << (r * 8) }

// ---------------------------------------------------------------------------
// The three atoms: set, clear, test a single bit.
// ---------------------------------------------------------------------------

func setBit(bb Bitboard, sq int) Bitboard    { return bb | (1 << sq) }
func clearBit(bb Bitboard, sq int) Bitboard  { return bb &^ (1 << sq) }
func isSet(bb Bitboard, sq int) bool         { return bb&(1<<sq) != 0 }
func popcount(bb Bitboard) int               { return bits.OnesCount64(uint64(bb)) }
func bitscan(bb Bitboard) int                { return bits.TrailingZeros64(uint64(bb)) }

// ---------------------------------------------------------------------------
// Position: 12 piece bitboards + derived occupancy bitboards.
// ---------------------------------------------------------------------------

type Position struct {
	WhitePawns, WhiteKnights, WhiteBishops, WhiteRooks, WhiteQueens, WhiteKing Bitboard
	BlackPawns, BlackKnights, BlackBishops, BlackRooks, BlackQueens, BlackKing Bitboard

	WhitePieces, BlackPieces, Occupied, Empty Bitboard
}

func (p *Position) UpdateDerived() {
	p.WhitePieces = p.WhitePawns | p.WhiteKnights | p.WhiteBishops |
		p.WhiteRooks | p.WhiteQueens | p.WhiteKing
	p.BlackPieces = p.BlackPawns | p.BlackKnights | p.BlackBishops |
		p.BlackRooks | p.BlackQueens | p.BlackKing
	p.Occupied = p.WhitePieces | p.BlackPieces
	p.Empty = ^p.Occupied
}

// ---------------------------------------------------------------------------
// Precomputed attack tables for non-sliders. Built ONCE at startup.
// knightAttacks[sq] = bitboard of every square a knight on sq can jump to.
// kingAttacks[sq]   = bitboard of every square a king on sq can step to.
// ---------------------------------------------------------------------------

var knightAttacks [64]Bitboard
var kingAttacks [64]Bitboard

// genStepper builds an attack table for a piece that moves by fixed offsets
// (knight: ±6,±10,±15,±17  |  king: ±1,±7,±8,±9). It checks that the target
// square is on the board AND that the file didn't wrap across the edge.
func genStepper(sq int, offsets [][2]int) Bitboard {
	var bb Bitboard
	rank, file := sq/8, sq%8
	for _, off := range offsets {
		dr, df := off[0], off[1]
		tr, tf := rank+dr, file+df
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

// ---------------------------------------------------------------------------
// Ray masks for classical sliders. Each square has 4 ray bitboards per
// sliding piece (rook: N/E/S/W  |  bishop: NE/NW/SE/SW), containing every
// square the slider COULD reach on that ray if nothing were in the way.
// The real attack set is computed at runtime by trimming the ray at the
// first blocker (classical method, no magics yet).
// ---------------------------------------------------------------------------

var rookRays [64][4]Bitboard // 0=N, 1=E, 2=S, 3=W
var bishopRays [64][4]Bitboard // 0=NE, 1=NW, 2=SE, 3=SW

func genRay(sq int, dr, df int) Bitboard {
	var bb Bitboard
	rank, file := sq/8, sq%8
	r, f := rank+dr, file+df
	for r >= 0 && r < 8 && f >= 0 && f < 8 {
		bb = setBit(bb, r*8+f)
		r += dr
		f += df
	}
	return bb
}

func init() {
	for sq := range 64 {
		rookRays[sq][0] = genRay(sq, 1, 0)  // north
		rookRays[sq][1] = genRay(sq, 0, 1)  // east
		rookRays[sq][2] = genRay(sq, -1, 0) // south
		rookRays[sq][3] = genRay(sq, 0, -1) // west
		bishopRays[sq][0] = genRay(sq, 1, 1)   // north-east
		bishopRays[sq][1] = genRay(sq, 1, -1)  // north-west
		bishopRays[sq][2] = genRay(sq, -1, 1)  // south-east
		bishopRays[sq][3] = genRay(sq, -1, -1) // south-west
	}
}

// rayPositive trims a ray that points toward HIGHER square indices (north,
// east, NE, NW) at the first blocker. The nearest blocker is the lowest set
// bit of (ray & occupied). The attack set is the ray up to and including
// that blocker; everything beyond is cut.
func rayPositive(ray, occupied Bitboard) Bitboard {
	blockers := ray & occupied
	if blockers == 0 {
		return ray
	}
	blocker := bitscan(blockers)
	// Keep bits 0..blocker (inclusive), cut everything strictly above.
	upToAndIncludingBlocker := (Bitboard(1) << (blocker + 1)) - 1
	return ray & upToAndIncludingBlocker
}

// rayNegative trims a ray that points toward LOWER square indices (south,
// west, SE, SW) at the first blocker. The nearest blocker is the highest
// set bit of (ray & occupied).
func rayNegative(ray, occupied Bitboard) Bitboard {
	blockers := ray & occupied
	if blockers == 0 {
		return ray
	}
	blocker := 63 - bits.LeadingZeros64(uint64(blockers))
	// Keep bits blocker..63 (inclusive), cut everything strictly below.
	fromBlockerUpwards := ^Bitboard(0) << blocker
	return ray & fromBlockerUpwards
}

// rookAttacks sums the 4 trimmed rays (N,E positive; S,W negative).
func rookAttacks(sq int, occupied Bitboard) Bitboard {
	return rayPositive(rookRays[sq][0], occupied) |
		rayPositive(rookRays[sq][1], occupied) |
		rayNegative(rookRays[sq][2], occupied) |
		rayNegative(rookRays[sq][3], occupied)
}

func bishopAttacks(sq int, occupied Bitboard) Bitboard {
	return rayPositive(bishopRays[sq][0], occupied) |
		rayPositive(bishopRays[sq][1], occupied) |
		rayNegative(bishopRays[sq][2], occupied) |
		rayNegative(bishopRays[sq][3], occupied)
}

// ---------------------------------------------------------------------------
// Move generators. Each returns a bitboard of TARGET squares, not moves yet.
// To turn targets into Move structs you'd bitscan them (see knightMoves).
// ---------------------------------------------------------------------------

// White pawn forward pushes: shift up one rank, keep only empty squares.
func whitePawnPushes(pawns, empty Bitboard) Bitboard {
	return (pawns << 8) & empty
}

// White pawn double pushes: a pawn on rank 2 may jump two squares if both
// the square in front and the destination are empty.
func whitePawnDoublePushes(pawns, empty Bitboard) Bitboard {
	single := whitePawnPushes(pawns, empty)            // can move to rank 3?
	double := (single << 8) & empty                     // and rank 4 is empty too?
	return double & rankMask(3)                         // only pawns that started on rank 2
}

// White pawn captures: two diagonal shifts, masked to avoid wrap-around.
func whitePawnCaptures(pawns, blackPieces Bitboard) Bitboard {
	left := ((pawns & notA) << 7) & blackPieces  // capture toward north-west (file -1)
	right := ((pawns & notH) << 9) & blackPieces // capture toward north-east (file +1)
	return left | right
}

// Knight moves: bitscan loop over each knight, OR its precomputed attacks,
// mask out own pieces (can't land on your own men).
func knightMoves(knights, ownPieces Bitboard) Bitboard {
	var moves Bitboard
	for knights != 0 {
		sq := bitscan(knights)
		knights &= knights - 1 // clear lowest set bit
		moves |= knightAttacks[sq] & ^ownPieces
	}
	return moves
}

// King moves: single king, so no loop — just one lookup.
func kingMoves(kingSq int, ownPieces Bitboard) Bitboard {
	return kingAttacks[kingSq] & ^ownPieces
}

// ---------------------------------------------------------------------------
// Printing.
// ---------------------------------------------------------------------------

func printBitboard(bb Bitboard, title string) {
	fmt.Println("---", title, "---")
	for rank := 7; rank >= 0; rank-- {
		fmt.Printf("%d ", rank+1)
		for file := range 8 {
			sq := rank*8 + file
			if isSet(bb, sq) {
				fmt.Print("1 ")
			} else {
				fmt.Print(". ")
			}
		}
		fmt.Println()
	}
	fmt.Println("  a b c d e f g h")
	fmt.Println("  popcount:", popcount(bb))
	fmt.Println()
}

// printBoard shows actual piece letters by combining the 12 bitboards.
func printBoard(p Position) {
	fmt.Println("--- position ---")
	for rank := 7; rank >= 0; rank-- {
		fmt.Printf("%d ", rank+1)
		for file := range 8 {
			sq := rank*8 + file
			switch {
			case isSet(p.WhitePawns, sq):
				fmt.Print("P ")
			case isSet(p.WhiteKnights, sq):
				fmt.Print("N ")
			case isSet(p.WhiteBishops, sq):
				fmt.Print("B ")
			case isSet(p.WhiteRooks, sq):
				fmt.Print("R ")
			case isSet(p.WhiteQueens, sq):
				fmt.Print("Q ")
			case isSet(p.WhiteKing, sq):
				fmt.Print("K ")
			case isSet(p.BlackPawns, sq):
				fmt.Print("p ")
			case isSet(p.BlackKnights, sq):
				fmt.Print("n ")
			case isSet(p.BlackBishops, sq):
				fmt.Print("b ")
			case isSet(p.BlackRooks, sq):
				fmt.Print("r ")
			case isSet(p.BlackQueens, sq):
				fmt.Print("q ")
			case isSet(p.BlackKing, sq):
				fmt.Print("k ")
			default:
				fmt.Print(". ")
			}
		}
		fmt.Println()
	}
	fmt.Println("  a b c d e f g h")
	fmt.Println()
}

// ---------------------------------------------------------------------------
// Main: set up a tiny position and verify each generator.
// ---------------------------------------------------------------------------

func main() {
	var p Position

	// White: pawns on a2 and e2, knight on g1, king on e1, rook on a1.
	p.WhitePawns = setBit(p.WhitePawns, 8)  // a2
	p.WhitePawns = setBit(p.WhitePawns, 12) // e2
	p.WhiteKnights = setBit(p.WhiteKnights, 6) // g1
	p.WhiteKing = setBit(p.WhiteKing, 4)       // e1
	p.WhiteRooks = setBit(p.WhiteRooks, 0)     // a1

	// Black: a pawn on d3 (so a white pawn can capture it), a pawn on e7.
	p.BlackPawns = setBit(p.BlackPawns, 19) // d3
	p.BlackPawns = setBit(p.BlackPawns, 52) // e7

	p.UpdateDerived()

	printBoard(p)
	printBitboard(p.WhitePieces, "whitePieces")
	printBitboard(p.Occupied, "occupied")
	printBitboard(p.Empty, "empty")

	// 1. Pawn pushes.
	printBitboard(whitePawnPushes(p.WhitePawns, p.Empty), "white pawn pushes (a2->a3, e2->e3)")

	// 2. Pawn double pushes: only the e2 pawn qualifies (a2 also does, both rank 2).
	printBitboard(whitePawnDoublePushes(p.WhitePawns, p.Empty), "white pawn double pushes (->rank 4)")

	// 3. Pawn captures: e2 pawn captures d3 (black pawn). a2 pawn has no captures.
	printBitboard(whitePawnCaptures(p.WhitePawns, p.BlackPieces), "white pawn captures (e2xd3)")

	// 4. Knight on g1 should reach e2, f3, h3 — but e2 is own pawn, so filtered out.
	printBitboard(knightMoves(p.WhiteKnights, p.WhitePieces), "white knight moves (g1 -> f3, h3)")

	// 5. King on e1 should reach d1,e2,d2,f1,f2,e2,d2 — own pieces filter some.
	printBitboard(kingMoves(4, p.WhitePieces), "white king moves (e1)")

	// 6. Sliders (classical, no magic). Rook on a1 sees a-file up to d3 (blocker).
	printBitboard(rookAttacks(0, p.Occupied), "rook on a1 attacks (classical ray trim)")

	// 7. Show a precomputed knight attack table entry for reference.
	printBitboard(knightAttacks[6], "knightAttacks[g1] (precomputed table)")
}