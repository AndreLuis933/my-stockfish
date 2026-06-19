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

	// undoStack records the state needed to reverse a Make call via Unmake.
	// Each Make pushes one undoInfo; each Unmake pops it. This is the standard
	// make/unmake pattern — far faster than copying the whole board per node.
	undoStack []undoInfo
}

// undoInfo holds exactly what Make overwrites, so Unmake can restore it.
// Storing the pre-move values here means we don't need to recompute them.
type undoInfo struct {
	captured         types.Piece // piece that was on the capture square (0 if none) — for en passant this is the captured pawn
	captureSquare    int         // where the captured piece was (== move.To, except en passant)
	enPassantCapture int         // pre-move EnPassantCapture
	enPassantTarget  int         // pre-move EnPassantTarget
	castlingRights   types.CastlingRights // pre-move castling rights
}

// Game is the single global Position used by the WASM bridge and the legacy
// free functions. The AI will later create its own *Position instances to
// search in parallel without touching Game.
var Game = &Position{
	EnPassantTarget:  -1,
	EnPassantCapture: -1,
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
	p.undoStack = p.undoStack[:0]
}