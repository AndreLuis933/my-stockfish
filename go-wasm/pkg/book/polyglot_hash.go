package book

import (
	"webassemble/pkg/engine"
	"webassemble/pkg/types"
)

// Polyglot piece type index: (pieceType-1)*2 + color, where color 0=black, 1=white.
// This matches python-chess's ZobristHasher which iterates occupied_co where
// pivot=0 corresponds to BLACK (False) and pivot=1 corresponds to WHITE (True).
//
//	wP=1, bP=0, wN=3, bN=2, wB=5, bB=4, wR=7, bR=6, wQ=9, bQ=8, wK=11, bK=10
var polyglotPieceIndex = [256]int{}

func init() {
	// Build the lookup table from Piece byte → polyglot piece index.
	// Polyglot index: (pieceType-1)*2 + color, where color 0=black, 1=white.
	// Engine Piece byte: type bits (one-hot) | color bits (6-7).
	//
	// Mapping (color 0=black, 1=white):
	//   bP=0, wP=1, bN=2, wN=3, bB=4, wB=5, bR=6, wR=7, bQ=8, wQ=9, bK=10, wK=11
	type pieceType struct {
		pt   types.Piece
		idxW int
		idxB int
	}
	pieceTypes := []pieceType{
		{types.Pawn, 1, 0},
		{types.Knight, 3, 2},
		{types.Bishop, 5, 4},
		{types.Rook, 7, 6},
		{types.Queen, 9, 8},
		{types.King, 11, 10},
	}
	for _, pt := range pieceTypes {
		polyglotPieceIndex[types.ColorWhite|pt.pt] = pt.idxW
		polyglotPieceIndex[types.ColorBlack|pt.pt] = pt.idxB
	}
}

// polyglotCastlingKeyIndex maps a CastlingRights bit to the polyglot key index
// (768 + offset). The polyglot order is: whiteK, whiteQ, blackK, blackQ.
var polyglotCastlingKeyIndex = map[types.CastlingRights]int{
	types.CastleWhiteK: 768,
	types.CastleWhiteQ: 769,
	types.CastleBlackK: 770,
	types.CastleBlackQ: 771,
}

// PolyglotHash computes the Polyglot Zobrist hash of a Position.
// This is independent of the engine's own Zobrist hash (different random constants).
//
// The hash includes:
//   - 768 piece keys (piece type × color × square)
//   - 4 castling keys (one per active right)
//   - 8 en passant keys (one per file, only if a pawn can actually capture ep)
//   - 1 side-to-move key (XOR if white to move — polyglot convention)
func PolyglotHash(p *engine.Position) uint64 {
	var hash uint64

	// 1) Piece keys
	for sq := range 64 {
		piece := p.Board[sq]
		if piece == 0 {
			continue
		}
		idx := polyglotPieceIndex[piece]
		if idx < 0 || idx >= 12 {
			continue
		}
		hash ^= polyglotKeys[64*idx+sq]
	}

	// 2) Castling keys
	for cr, offset := range polyglotCastlingKeyIndex {
		if p.CastlingRights&cr != 0 {
			hash ^= polyglotKeys[offset]
		}
	}

	// 3) En passant key — only if a pawn of the side to move can actually capture ep.
	//    This matches the polyglot reference and python-chess behavior.
	if p.EnPassantTarget >= 0 {
		file := p.EnPassantTarget % 8
		if canCaptureEnPassant(p, p.EnPassantTarget) {
			hash ^= polyglotKeys[772+file]
		}
	}

	// 4) Side-to-move key (polyglot XORs if white to move)
	if p.WhiteToMove {
		hash ^= polyglotKeys[780]
	}

	return hash
}

// canCaptureEnPassant checks if a pawn of the side to move is adjacent to
// the en passant target square and could capture it. This matches the
// polyglot convention where the ep key is only included if a capture is
// actually possible (not just because the ep square is set).
func canCaptureEnPassant(p *engine.Position, epSquare int) bool {
	if epSquare < 0 || epSquare >= 64 {
		return false
	}

	file := epSquare % 8
	rank := epSquare / 8

	var pawnRank int
	var pawnColor types.Piece
	if p.WhiteToMove {
		// White to move: ep square is on rank 6 (index 5), capturing pawn on rank 5 (index 4)
		if rank != 5 {
			return false
		}
		pawnRank = 4
		pawnColor = types.ColorWhite
	} else {
		// Black to move: ep square is on rank 3 (index 2), capturing pawn on rank 4 (index 3)
		if rank != 2 {
			return false
		}
		pawnRank = 3
		pawnColor = types.ColorBlack
	}

	pawnPiece := pawnColor | types.Pawn

	// Check left neighbor (file - 1)
	if file > 0 {
		leftSq := pawnRank*8 + file - 1
		if p.Board[leftSq] == pawnPiece {
			return true
		}
	}

	// Check right neighbor (file + 1)
	if file < 7 {
		rightSq := pawnRank*8 + file + 1
		if p.Board[rightSq] == pawnPiece {
			return true
		}
	}

	return false
}