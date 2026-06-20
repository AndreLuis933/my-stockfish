package engine

import (
	"math/rand"

	"webassemble/pkg/types"
)

// Zobrist hashing: XOR of random 64-bit keys for each (piece, square) pair,
// plus keys for side-to-move, castling rights, and en passant file.
//
// The hash is maintained incrementally in Make/Unmake (XOR is its own inverse),
// so it costs ~5-10 extra XOR ops per move — far cheaper than recomputing.

var (
	zobristPiece  [12][64]uint64 // [pieceIndex][square]
	zobristSide   uint64         // XOR when black to move
	zobristCastle [16]uint64     // [castlingRights bitmask 0-15]
	zobristEP     [8]uint64      // [en passant file 0-7]
)

// pieceZobristIndex maps a Piece (type bits + color bits) to a 0-11 index.
// White: Pawn=0, Knight=1, Bishop=2, Rook=3, Queen=4, King=5
// Black: Pawn=6, Knight=7, Bishop=8, Rook=9, Queen=10, King=11
func pieceZobristIndex(p types.Piece) int {
	colorOffset := 0
	if p&types.ColorMask == types.ColorBlack {
		colorOffset = 6
	}
	switch p & types.TypeMask {
	case types.Pawn:
		return colorOffset + 0
	case types.Knight:
		return colorOffset + 1
	case types.Bishop:
		return colorOffset + 2
	case types.Rook:
		return colorOffset + 3
	case types.Queen:
		return colorOffset + 4
	case types.King:
		return colorOffset + 5
	}
	return -1
}

// initZobrist generates the random keys using a fixed seed for reproducibility.
// The same seed guarantees identical hashes across runs, so TT entries remain
// valid between games (if the TT isn't cleared).
func initZobrist() {
	r := rand.New(rand.NewSource(0xCAFE))
	for i := range zobristPiece {
		for j := range zobristPiece[i] {
			zobristPiece[i][j] = r.Uint64()
		}
	}
	zobristSide = r.Uint64()
	for i := range zobristCastle {
		zobristCastle[i] = r.Uint64()
	}
	for i := range zobristEP {
		zobristEP[i] = r.Uint64()
	}
}

func init() {
	initZobrist()
}

// ComputeHash computes the full Zobrist hash for a position from scratch.
// Called once in LoadFen; maintained incrementally by Make/Unmake after that.
func (p *Position) ComputeHash() uint64 {
	var h uint64
	for sq, piece := range p.Board {
		if piece == 0 {
			continue
		}
		idx := pieceZobristIndex(piece)
		if idx >= 0 {
			h ^= zobristPiece[idx][sq]
		}
	}
	if !p.WhiteToMove {
		h ^= zobristSide
	}
	h ^= zobristCastle[p.CastlingRights]
	if p.EnPassantTarget >= 0 {
		file := p.EnPassantTarget % 8
		h ^= zobristEP[file]
	}
	return h
}

// hashDeltaMove returns the XOR delta to apply when making a move (excluding
// the captured piece and castling/EP changes, which are handled by callers).
// XOR is its own inverse, so the same delta is used for Make and Unmake.
func hashDeltaMove(piece types.Piece, from, to int) uint64 {
	idx := pieceZobristIndex(piece)
	return zobristPiece[idx][from] ^ zobristPiece[idx][to]
}

func hashDeltaPiece(piece types.Piece, sq int) uint64 {
	idx := pieceZobristIndex(piece)
	return zobristPiece[idx][sq]
}