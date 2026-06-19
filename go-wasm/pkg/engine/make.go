package engine

import "webassemble/pkg/types"

// MakeMove applies a move (given as raw from/to/promotion) to the global Game.
// Kept as a free function for the WASM bridge, which receives primitive args
// from JavaScript. Internally it builds a Move and delegates to Make, which
// reads the flag/captured fields set by the move generators.
func MakeMove(from, to, promotion int) {
	piece := Game.Board[from]
	move := types.Move{From: from, To: to}

	// Infer the flag from the board (the bridge doesn't know move flags).
	// This path is only used by the frontend; the engine and AI always use
	// Make(move) directly with a fully-populated move from the generators.
	switch {
	case piece&types.Pawn != 0 && promotion != 0:
		move.Flag = types.FlagPromotion
		move.Promotion = PiecePtr(types.Piece(promotion))
	case piece&types.King != 0 && abs(to-from) == 2:
		if to > from {
			move.Flag = types.FlagCastleK
		} else {
			move.Flag = types.FlagCastleQ
		}
	case piece&types.Pawn != 0 && abs(to-from) == 2*boardSize:
		move.Flag = types.FlagDoublePush
	case piece&types.Pawn != 0 && to == Game.EnPassantTarget && Game.EnPassantCapture != -1:
		move.Flag = types.FlagEnPassant
		move.Captured = Game.Board[Game.EnPassantCapture]
	default:
		if Game.Board[to] != 0 {
			move.Captured = Game.Board[to]
		}
	}

	Game.Make(move)
}

// Make applies a fully-populated move to this position and pushes undo info
// onto the undo stack so Unmake can reverse it.
//
// It handles, based on move.Flag:
//   - FlagNormal:      move the piece, capture if Captured != 0
//   - FlagDoublePush:  move the pawn, set en passant target for next move
//   - FlagEnPassant:   move the pawn, remove the captured pawn (not on `to`)
//   - FlagCastleK/Q:   move the king two squares, move the rook across it
//   - FlagPromotion:   replace the pawn with the promoted piece
//
// Castling rights are updated when the king or a rook moves, or a rook is
// captured on its origin corner. The side to move is flipped at the end.
func (p *Position) Make(move types.Move) {
	from, to := move.From, move.To
	piece := p.Board[from]

	// Record where the captured piece actually sits. For all captures except
	// en passant, that's `to`. For en passant, the captured pawn is on a
	// different square (EnPassantCapture).
	captureSquare := to
	if move.Flag == types.FlagEnPassant {
		captureSquare = p.EnPassantCapture
	}

	// Push undo info so Unmake can restore everything Make changes.
	p.undoStack = append(p.undoStack, undoInfo{
		captured:         move.Captured,
		captureSquare:    captureSquare,
		enPassantCapture: p.EnPassantCapture,
		enPassantTarget:  p.EnPassantTarget,
		castlingRights:   p.CastlingRights,
	})

	// Clear en passant state every move; only DoublePush sets a new one.
	// (The old value is already saved in undoInfo above.)
	p.EnPassantCapture, p.EnPassantTarget = -1, -1

	switch move.Flag {
	case types.FlagEnPassant:
		// Remove the captured pawn (sits on the old EnPassantCapture square),
		// move our pawn to `to`. The `to` square was empty.
		p.Board[captureSquare] = 0
		p.Board[from] = 0
		p.Board[to] = piece

	case types.FlagDoublePush:
		p.Board[from] = 0
		p.Board[to] = piece
		// The square the pawn skipped over is the e.p. target; the pawn's
		// new square is where an enemy pawn would capture from.
		p.EnPassantCapture = to
		p.EnPassantTarget = (from + to) / 2

	case types.FlagCastleK:
		p.Board[from] = 0
		p.Board[to] = piece
		// Rook from h-file to f-file (king's right).
		rook := p.Board[to+1]
		p.Board[to+1] = 0
		p.Board[to-1] = rook

	case types.FlagCastleQ:
		p.Board[from] = 0
		p.Board[to] = piece
		// Rook from a-file to d-file (king's left).
		rook := p.Board[to-2]
		p.Board[to-2] = 0
		p.Board[to+1] = rook

	case types.FlagPromotion:
		p.Board[from] = 0
		if move.Promotion != nil {
			p.Board[to] = *move.Promotion
		} else {
			// Fallback: promotion move built without a Promotion piece
			// (e.g. from the raw bridge path) — default to Queen.
			p.Board[to] = piece | types.Queen
		}

	default: // FlagNormal
		p.Board[from] = 0
		p.Board[to] = piece
	}

	// Castling rights — king move clears that color's rights.
	if piece&types.King == types.King {
		if piece.Color() == types.ColorWhite {
			p.CastlingRights &^= types.CastleWhiteAll
		} else {
			p.CastlingRights &^= types.CastleBlackAll
		}
	}

	// Castling rights — rook moves from origin, or rook captured on origin.
	if piece&types.Rook == types.Rook || move.Captured&types.Rook == types.Rook {
		switch {
		case from == 0 || to == 0:
			p.CastlingRights &^= types.CastleWhiteQ
		case from == 7 || to == 7:
			p.CastlingRights &^= types.CastleWhiteK
		case from == 56 || to == 56:
			p.CastlingRights &^= types.CastleBlackQ
		case from == 63 || to == 63:
			p.CastlingRights &^= types.CastleBlackK
		}
	}

	p.WhiteToMove = !p.WhiteToMove
}

// Unmake reverses the most recent Make call, restoring the position to its
// exact pre-move state. It pops from the undo stack.
//
// For each flag it undoes the board changes, then restores the saved en
// passant / castling state. The moving piece is recovered from `to` (for
// promotions, the pawn is recovered by stripping the promoted type bits and
// keeping the pawn type + color).
func (p *Position) Unmake(move types.Move) {
	from, to := move.From, move.To

	// Flip side to move first — we're now undoing the move of the side that
	// just moved, so the side to move goes back to them.
	p.WhiteToMove = !p.WhiteToMove

	// Pop undo info from the stack.
	n := len(p.undoStack) - 1
	undo := p.undoStack[n]
	p.undoStack = p.undoStack[:n]

	// Restore the moving piece to its origin. For promotions, the piece on
	// `to` is the promoted piece — we need to put the original pawn back on
	// `from`. The pawn's color is the same as the promoted piece's color.
	switch move.Flag {
	case types.FlagEnPassant:
		// Move our pawn back from `to` to `from`, clear `to` (it was empty),
		// and put the captured pawn back on its square.
		pawn := p.Board[to]
		p.Board[from] = pawn
		p.Board[to] = 0
		p.Board[undo.captureSquare] = undo.captured

	case types.FlagCastleK:
		// Move king back, move rook from f-file back to h-file.
		p.Board[from] = p.Board[to]
		p.Board[to] = 0
		rook := p.Board[to-1]
		p.Board[to-1] = 0
		p.Board[to+1] = rook

	case types.FlagCastleQ:
		// Move king back, move rook from d-file back to a-file.
		p.Board[from] = p.Board[to]
		p.Board[to] = 0
		rook := p.Board[to+1]
		p.Board[to+1] = 0
		p.Board[to-2] = rook

	case types.FlagPromotion:
		// The piece on `to` is the promoted piece. Reconstruct the pawn:
		// keep the color bits, set the type to Pawn.
		color := p.Board[to] & types.ColorMask
		p.Board[from] = color | types.Pawn
		// Restore captured piece on `to` (or clear it if none).
		p.Board[to] = undo.captured

	default: // FlagNormal, FlagDoublePush
		p.Board[from] = p.Board[to]
		p.Board[to] = undo.captured
	}

	// Restore the pre-move en passant and castling state.
	p.EnPassantCapture = undo.enPassantCapture
	p.EnPassantTarget = undo.enPassantTarget
	p.CastlingRights = undo.castlingRights
}