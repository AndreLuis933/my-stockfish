package engine

import "webassemble/pkg/types"

// Position holds the full state of a chess game at a single moment.
// Every function that needs the board works on *Position (as a method receiver)
// instead of reading package-level globals. This makes the engine reusable,
// testable in parallel, and ready for AI search (which explores many positions).
type Position struct {
	Board            types.Board
	WhiteToMove      bool
	CastlingRights   types.CastlingRights
	EnPassantTarget  int // square behind the pawn that just did a double push; -1 if none
	EnPassantCapture int // square of the pawn that can be captured e.p.; -1 if none
	HalfmoveClock    int // plies since last pawn move or capture (for 50-move rule)
	FullmoveNumber   int // increments after black's move

	// Hash is the Zobrist hash of the full position state, maintained
	// incrementally by Make/Unmake. Used by the transposition table.
	Hash uint64

	// KingSquares caches the king positions for O(1) check detection.
	// Updated incrementally in Make/Unmake. -1 means "not on board".
	KingSquares [2]int // [white=0, black=1]

	// EvalScore is the incremental static evaluation (white material+PST minus
	// black material+PST), maintained by Make/Unmake. Evaluate() reads this
	// and negates for the side-to-move perspective.
	EvalScore int

	// Bitboards (hybrid: kept alongside the mailbox). The 12 piece bitboards
	// are maintained incrementally by Make/Unmake. The 4 derived occupancy
	// bitboards are recomputed after each update. Used by move generation and
	// attack detection (Phase 2).
	WhitePawns, WhiteKnights, WhiteBishops, WhiteRooks, WhiteQueens, WhiteKing Bitboard
	BlackPawns, BlackKnights, BlackBishops, BlackRooks, BlackQueens, BlackKing Bitboard
	WhitePieces, BlackPieces, Occupied, Empty Bitboard
	dummyBB Bitboard // safety fallback for pieceBitboardFor(0) — never written to intentionally

	// undoStack is a fixed-size stack of undo info, enabling O(1) Make/Unmake
	// with zero heap allocation. 256 is well beyond any realistic search depth.
	undoStack [maxPly]undoInfo
	undoPly   int
}

const maxPly = 512

// undoInfo holds exactly what Make overwrites, so Unmake can restore it.
// Storing the pre-move values here means we don't need to recompute them.
type undoInfo struct {
	captured         types.Piece
	captureSquare    int
	enPassantCapture int
	enPassantTarget  int
	castlingRights   types.CastlingRights
	halfmoveClock    int
	evalScore        int
	hash             uint64
}

// Game is the single global Position used by the WASM bridge and the legacy
// free functions. The AI will later create its own *Position instances to
// search in parallel without touching Game.
var Game = &Position{
	EnPassantTarget:  -1,
	EnPassantCapture: -1,
	KingSquares:      [2]int{-1, -1},
}

func (p *Position) Ply() int { return p.undoPly }

// updateBitboards scans the mailbox Board and populates the 12 piece
// bitboards plus the 4 derived occupancy bitboards. Called after LoadFen
// (full rebuild) and in tests to verify Make/Unmake keeps bitboards in sync.
func (p *Position) updateBitboards() {
	p.WhitePawns, p.WhiteKnights, p.WhiteBishops, p.WhiteRooks, p.WhiteQueens, p.WhiteKing = 0, 0, 0, 0, 0, 0
	p.BlackPawns, p.BlackKnights, p.BlackBishops, p.BlackRooks, p.BlackQueens, p.BlackKing = 0, 0, 0, 0, 0, 0

	for sq, piece := range p.Board {
		if piece == 0 {
			continue
		}
		bb := p.pieceBitboardFor(piece)
		*bb |= Bitboard(1) << sq
	}

	p.recomputeOccupancy()
}

// recomputeOccupancy derives WhitePieces, BlackPieces, Occupied, and Empty
// from the 12 piece bitboards. Called after LoadFen (full rebuild).
func (p *Position) recomputeOccupancy() {
	p.WhitePieces = p.WhitePawns | p.WhiteKnights | p.WhiteBishops |
		p.WhiteRooks | p.WhiteQueens | p.WhiteKing
	p.BlackPieces = p.BlackPawns | p.BlackKnights | p.BlackBishops |
		p.BlackRooks | p.BlackQueens | p.BlackKing
	p.Occupied = p.WhitePieces | p.BlackPieces
	p.Empty = ^p.Occupied
}

// xorOccupancy applies a delta to the global occupancy bitboards.
func (p *Position) xorOccupancy(delta Bitboard) {
	p.Occupied ^= delta
	p.Empty = ^p.Occupied
}

// movePieceBB moves a bit from→to in the piece bitboard, XORs the color
// occupancy, and returns the delta for global occupancy update.
func (p *Position) movePieceBB(bb *Bitboard, colorOcc *Bitboard, from, to int) Bitboard {
	delta := Bitboard(1)<<from | Bitboard(1)<<to
	*bb ^= delta
	*colorOcc ^= delta
	return delta
}

// removePieceBB clears a bit from the piece bitboard and color occupancy.
// Returns the delta for global occupancy update.
func (p *Position) removePieceBB(bb *Bitboard, colorOcc *Bitboard, sq int) Bitboard {
	delta := Bitboard(1) << sq
	*bb &^= delta
	*colorOcc &^= delta
	return delta
}

// addPieceBB sets a bit in the piece bitboard and color occupancy.
// Returns the delta for global occupancy update.
func (p *Position) addPieceBB(bb *Bitboard, colorOcc *Bitboard, sq int) Bitboard {
	delta := Bitboard(1) << sq
	*bb |= delta
	*colorOcc |= delta
	return delta
}

// colorOccupancy returns a pointer to WhitePieces or BlackPieces based on the
// piece's color.
func (p *Position) colorOccupancy(piece types.Piece) *Bitboard {
	if piece.Color() == types.ColorWhite {
		return &p.WhitePieces
	}
	return &p.BlackPieces
}

// pieceBitboardFor returns a pointer to the bitboard field corresponding to
// the given piece (type + color). Used by updateBitboards and Make/Unmake
// to avoid a 12-way switch at each call site. Returns a dummy bitboard for
// empty pieces (piece=0) so callers can safely dereference without nil checks.
func (p *Position) pieceBitboardFor(piece types.Piece) *Bitboard {
	pt := piece & types.TypeMask
	switch piece & types.ColorMask {
	case types.ColorWhite:
		switch pt {
		case types.Pawn:
			return &p.WhitePawns
		case types.Knight:
			return &p.WhiteKnights
		case types.Bishop:
			return &p.WhiteBishops
		case types.Rook:
			return &p.WhiteRooks
		case types.Queen:
			return &p.WhiteQueens
		case types.King:
			return &p.WhiteKing
		}
	case types.ColorBlack:
		switch pt {
		case types.Pawn:
			return &p.BlackPawns
		case types.Knight:
			return &p.BlackKnights
		case types.Bishop:
			return &p.BlackBishops
		case types.Rook:
			return &p.BlackRooks
		case types.Queen:
			return &p.BlackQueens
		case types.King:
			return &p.BlackKing
		}
	}
	return &p.dummyBB
}

// TrimUndoStack keeps only the last n entries of the undo stack and resets
// undoPly to n. This is used by the AI search to start with a near-full
// 256-ply budget regardless of game length, while keeping enough recent
// history for short-cycle repetition detection (perpetual checks, shuffles).
// If the stack already has n or fewer entries, this is a no-op.
func (p *Position) TrimUndoStack(n int) {
	if p.undoPly <= n {
		return
	}
	offset := p.undoPly - n
	copy(p.undoStack[:n], p.undoStack[offset:p.undoPly])
	p.undoPly = n
}

// reset empties the position (used by tests and before LoadFen).
func (p *Position) reset() {
	p.Board = types.Board{}
	p.WhiteToMove = false
	p.CastlingRights = 0
	p.EnPassantTarget = -1
	p.EnPassantCapture = -1
	p.HalfmoveClock = 0
	p.FullmoveNumber = 0
	p.KingSquares = [2]int{-1, -1}
	p.EvalScore = 0
	p.Hash = 0
	p.undoPly = 0

	p.WhitePawns, p.WhiteKnights, p.WhiteBishops, p.WhiteRooks, p.WhiteQueens, p.WhiteKing = 0, 0, 0, 0, 0, 0
	p.BlackPawns, p.BlackKnights, p.BlackBishops, p.BlackRooks, p.BlackQueens, p.BlackKing = 0, 0, 0, 0, 0, 0
	p.WhitePieces, p.BlackPieces, p.Occupied, p.Empty = 0, 0, 0, 0
}