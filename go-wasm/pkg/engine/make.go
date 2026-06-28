package engine

import "webassemble/pkg/types"

// searchBudgetMargin is the ply headroom reserved for the AI search when
// deciding to trim the real Game's undo stack in MakeMove. It covers the
// worst-case search depth (maxDepth) plus check extensions and quiescence,
// so the search always has room to run without hitting the ply guard.
const searchBudgetMargin = 100

// MakeMove applies a move (given as raw from/to/promotion) to the global Game.
// Kept as a free function for the WASM bridge, which receives primitive args
// from JavaScript. Internally it builds a Move and delegates to Make, which
// reads the flag/captured fields set by the move generators.
func MakeMove(from, to, promotion int) {
	// Dynamic trim: keep the undo stack bounded so the AI search always has
	// a full ply budget regardless of game length. Trims to HalfmoveClock+1
	// (the full reversible-move window) so threefold repetition detection in
	// the real game stays correct. Only fires on long games (200+ moves);
	// normal games never reach this threshold. MakeMove is never called by
	// the search, so this can never fire mid-search.
	if Game.undoPly > maxPly-searchBudgetMargin {
		Game.TrimUndoStack(Game.HalfmoveClock + 1)
	}

	piece := Game.Board[from]
	move := types.Move{From: from, To: to}

	// Infer the flag from the board (the bridge doesn't know move flags).
	// This path is only used by the frontend; the engine and AI always use
	// Make(move) directly with a fully-populated move from the generators.
	switch {
	case piece&types.Pawn != 0 && promotion != 0:
		move.Flag = types.FlagPromotion
		move.Promotion = types.Piece(promotion)
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
	p.undoStack[p.undoPly] = undoInfo{
		captured:         move.Captured,
		captureSquare:    captureSquare,
		enPassantCapture: p.EnPassantCapture,
		enPassantTarget:  p.EnPassantTarget,
		castlingRights:   p.CastlingRights,
		halfmoveClock:    p.HalfmoveClock,
		evalScore:        p.EvalScore,
		hash:             p.Hash,
	}
	p.undoPly++

	// Save old state for hash delta computation.
	oldCastle := p.CastlingRights
	oldEPFile := -1
	if p.EnPassantTarget >= 0 {
		oldEPFile = p.EnPassantTarget % 8
	}

	// Clear en passant state every move; only DoublePush sets a new one.
	// (The old value is already saved in undoInfo above.)
	p.EnPassantCapture, p.EnPassantTarget = -1, -1

	// Incremental evaluation + hash: adjust EvalScore and Hash for every piece
	// that moves or is captured. signedPieceValue returns +val for white,
	// -val for black, so removing a piece subtracts its contribution and adding
	// a piece adds it. Hash uses XOR (its own inverse).
	var evalDelta int
	var hashDelta uint64

	var occDelta Bitboard
	isWhite := piece.Color() == types.ColorWhite
	var colorOcc *Bitboard
	if isWhite {
		colorOcc = &p.WhitePieces
	} else {
		colorOcc = &p.BlackPieces
	}

	switch move.Flag {
	case types.FlagEnPassant:
		p.Board[captureSquare] = 0
		p.Board[from] = 0
		p.Board[to] = piece
		// Inline: move piece bitboard from→to, clear captured.
		moveBB := p.pieceBitboardFor(piece)
		delta := Bitboard(1)<<from | Bitboard(1)<<to
		*moveBB ^= delta
		*colorOcc ^= delta
		occDelta ^= delta
		capBB := p.pieceBitboardFor(move.Captured)
		capDelta := Bitboard(1) << captureSquare
		*capBB &^= capDelta
		capColorOcc := p.colorOccupancy(move.Captured)
		*capColorOcc &^= capDelta
		occDelta ^= capDelta
		evalDelta -= signedPieceValue(move.Captured, captureSquare)
		evalDelta -= signedPieceValue(piece, from)
		evalDelta += signedPieceValue(piece, to)
		hashDelta ^= hashDeltaPiece(move.Captured, captureSquare)
		hashDelta ^= hashDeltaMove(piece, from, to)

	case types.FlagDoublePush:
		p.Board[from] = 0
		p.Board[to] = piece
		p.EnPassantCapture = to
		p.EnPassantTarget = (from + to) / 2
		moveBB := p.pieceBitboardFor(piece)
		delta := Bitboard(1)<<from | Bitboard(1)<<to
		*moveBB ^= delta
		*colorOcc ^= delta
		occDelta ^= delta
		evalDelta -= signedPieceValue(piece, from)
		evalDelta += signedPieceValue(piece, to)
		hashDelta ^= hashDeltaMove(piece, from, to)

	case types.FlagCastleK:
		p.Board[from] = 0
		p.Board[to] = piece
		rook := p.Board[to+1]
		p.Board[to+1] = 0
		p.Board[to-1] = rook
		rookFrom, rookTo := to+1, to-1
		moveBB := p.pieceBitboardFor(piece)
		delta := Bitboard(1)<<from | Bitboard(1)<<to
		*moveBB ^= delta
		*colorOcc ^= delta
		occDelta ^= delta
		rookBB := p.pieceBitboardFor(rook)
		rookDelta := Bitboard(1)<<rookFrom | Bitboard(1)<<rookTo
		*rookBB ^= rookDelta
		*colorOcc ^= rookDelta
		occDelta ^= rookDelta
		evalDelta -= signedPieceValue(piece, from)
		evalDelta += signedPieceValue(piece, to)
		evalDelta -= signedPieceValue(rook, rookFrom)
		evalDelta += signedPieceValue(rook, rookTo)
		hashDelta ^= hashDeltaMove(piece, from, to)
		hashDelta ^= hashDeltaMove(rook, rookFrom, rookTo)

	case types.FlagCastleQ:
		p.Board[from] = 0
		p.Board[to] = piece
		rook := p.Board[to-2]
		p.Board[to-2] = 0
		p.Board[to+1] = rook
		rookFrom, rookTo := to-2, to+1
		moveBB := p.pieceBitboardFor(piece)
		delta := Bitboard(1)<<from | Bitboard(1)<<to
		*moveBB ^= delta
		*colorOcc ^= delta
		occDelta ^= delta
		rookBB := p.pieceBitboardFor(rook)
		rookDelta := Bitboard(1)<<rookFrom | Bitboard(1)<<rookTo
		*rookBB ^= rookDelta
		*colorOcc ^= rookDelta
		occDelta ^= rookDelta
		evalDelta -= signedPieceValue(piece, from)
		evalDelta += signedPieceValue(piece, to)
		evalDelta -= signedPieceValue(rook, rookFrom)
		evalDelta += signedPieceValue(rook, rookTo)
		hashDelta ^= hashDeltaMove(piece, from, to)
		hashDelta ^= hashDeltaMove(rook, rookFrom, rookTo)

	case types.FlagPromotion:
		p.Board[from] = 0
		var promoPiece types.Piece
		if move.Promotion != 0 {
			promoPiece = move.Promotion
		} else {
			promoPiece = piece | types.Queen
		}
		p.Board[to] = promoPiece
		// Clear pawn at from.
		fromDelta := Bitboard(1) << from
		*p.pieceBitboardFor(piece) &^= fromDelta
		*colorOcc &^= fromDelta
		occDelta ^= fromDelta
		// Set promoted piece at to.
		toDelta := Bitboard(1) << to
		*p.pieceBitboardFor(promoPiece) |= toDelta
		*colorOcc |= toDelta
		occDelta ^= toDelta
		// Clear captured if any.
		if move.Captured != 0 {
			capColorOcc := p.colorOccupancy(move.Captured)
			*p.pieceBitboardFor(move.Captured) &^= toDelta
			*capColorOcc &^= toDelta
			occDelta ^= toDelta
		}
		evalDelta -= signedPieceValue(piece, from)
		evalDelta += signedPieceValue(promoPiece, to)
		if move.Captured != 0 {
			evalDelta -= signedPieceValue(move.Captured, to)
		}
		hashDelta ^= hashDeltaPiece(piece, from)
		hashDelta ^= hashDeltaPiece(promoPiece, to)
		if move.Captured != 0 {
			hashDelta ^= hashDeltaPiece(move.Captured, to)
		}

	default: // FlagNormal
		p.Board[from] = 0
		p.Board[to] = piece
		moveBB := p.pieceBitboardFor(piece)
		delta := Bitboard(1)<<from | Bitboard(1)<<to
		*moveBB ^= delta
		*colorOcc ^= delta
		occDelta ^= delta
		if move.Captured != 0 {
			capColorOcc := p.colorOccupancy(move.Captured)
			capBB := p.pieceBitboardFor(move.Captured)
			*capBB &^= Bitboard(1) << to
			*capColorOcc &^= Bitboard(1) << to
			occDelta ^= Bitboard(1) << to
		}
		evalDelta -= signedPieceValue(piece, from)
		evalDelta += signedPieceValue(piece, to)
		if move.Captured != 0 {
			evalDelta -= signedPieceValue(move.Captured, to)
		}
		hashDelta ^= hashDeltaMove(piece, from, to)
		if move.Captured != 0 {
			hashDelta ^= hashDeltaPiece(move.Captured, to)
		}
	}

	p.Occupied ^= occDelta
	p.Empty = ^p.Occupied

	p.EvalScore += evalDelta

	// Update castling rights and track if they changed (for hash).
	if piece&types.King == types.King {
		if piece.Color() == types.ColorWhite {
			p.CastlingRights &^= types.CastleWhiteAll
			p.KingSquares[0] = to
		} else {
			p.CastlingRights &^= types.CastleBlackAll
			p.KingSquares[1] = to
		}
	}

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

	// Hash: castling rights change.
	if oldCastle != p.CastlingRights {
		hashDelta ^= zobristCastle[oldCastle]
		hashDelta ^= zobristCastle[p.CastlingRights]
	}

	// Hash: en passant target change.
	if oldEPFile >= 0 {
		hashDelta ^= zobristEP[oldEPFile]
	}
	if p.EnPassantTarget >= 0 {
		hashDelta ^= zobristEP[p.EnPassantTarget%8]
	}

	// Hash: side to move flip.
	hashDelta ^= zobristSide

	// Halfmove clock — resets to 0 on pawn moves and captures, else increments.
	// The 50-move rule: if this clock reaches 100 (50 full moves without a
	// pawn move or capture), the game is a draw. The AI needs this to avoid
	// grinding out dead positions.
	isPawnMove := piece&types.Pawn == types.Pawn
	isCapture := move.Captured != 0 || move.Flag == types.FlagEnPassant
	if isPawnMove || isCapture {
		p.HalfmoveClock = 0
	} else {
		p.HalfmoveClock++
	}

	// Fullmove number — increments after black moves (i.e., when black was
	// the side that just moved, which means WhiteToMove was false before the
	// flip below). We check the pre-flip state.
	if !p.WhiteToMove {
		p.FullmoveNumber++
	}

	p.WhiteToMove = !p.WhiteToMove
	p.Hash ^= hashDelta
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
	p.undoPly--
	undo := p.undoStack[p.undoPly]

	// Restore the moving piece to its origin. For promotions, the piece on
	// `to` is the promoted piece — we need to put the original pawn back on
	// `from`. The pawn's color is the same as the promoted piece's color.
	var occDelta Bitboard

	switch move.Flag {
	case types.FlagEnPassant:
		pawn := p.Board[to]
		p.Board[from] = pawn
		p.Board[to] = 0
		p.Board[undo.captureSquare] = undo.captured
		pawnBB := p.pieceBitboardFor(pawn)
		pawnColorOcc := p.colorOccupancy(pawn)
		delta := Bitboard(1)<<from | Bitboard(1)<<to
		*pawnBB ^= delta
		*pawnColorOcc ^= delta
		occDelta ^= delta
		capDelta := Bitboard(1) << undo.captureSquare
		*p.pieceBitboardFor(undo.captured) |= capDelta
		*p.colorOccupancy(undo.captured) |= capDelta
		occDelta ^= capDelta

	case types.FlagCastleK:
		p.Board[from] = p.Board[to]
		p.Board[to] = 0
		rook := p.Board[to-1]
		p.Board[to-1] = 0
		p.Board[to+1] = rook
		rookFrom, rookTo := to+1, to-1
		king := p.Board[from]
		kingBB := p.pieceBitboardFor(king)
		kingColorOcc := p.colorOccupancy(king)
		kingDelta := Bitboard(1)<<from | Bitboard(1)<<to
		*kingBB ^= kingDelta
		*kingColorOcc ^= kingDelta
		occDelta ^= kingDelta
		rookBB := p.pieceBitboardFor(rook)
		rookColorOcc := p.colorOccupancy(rook)
		rookDelta := Bitboard(1)<<rookFrom | Bitboard(1)<<rookTo
		*rookBB ^= rookDelta
		*rookColorOcc ^= rookDelta
		occDelta ^= rookDelta

	case types.FlagCastleQ:
		p.Board[from] = p.Board[to]
		p.Board[to] = 0
		rook := p.Board[to+1]
		p.Board[to+1] = 0
		p.Board[to-2] = rook
		rookFrom, rookTo := to-2, to+1
		king := p.Board[from]
		kingBB := p.pieceBitboardFor(king)
		kingColorOcc := p.colorOccupancy(king)
		kingDelta := Bitboard(1)<<from | Bitboard(1)<<to
		*kingBB ^= kingDelta
		*kingColorOcc ^= kingDelta
		occDelta ^= kingDelta
		rookBB := p.pieceBitboardFor(rook)
		rookColorOcc := p.colorOccupancy(rook)
		rookDelta := Bitboard(1)<<rookFrom | Bitboard(1)<<rookTo
		*rookBB ^= rookDelta
		*rookColorOcc ^= rookDelta
		occDelta ^= rookDelta

	case types.FlagPromotion:
		color := p.Board[to] & types.ColorMask
		pawn := color | types.Pawn
		promoPiece := p.Board[to]
		p.Board[from] = pawn
		p.Board[to] = undo.captured
		promoColorOcc := p.colorOccupancy(promoPiece)
		toDelta := Bitboard(1) << to
		*p.pieceBitboardFor(promoPiece) &^= toDelta
		*promoColorOcc &^= toDelta
		occDelta ^= toDelta
		fromDelta := Bitboard(1) << from
		*p.pieceBitboardFor(pawn) |= fromDelta
		*p.colorOccupancy(pawn) |= fromDelta
		occDelta ^= fromDelta
		if undo.captured != 0 {
			*p.pieceBitboardFor(undo.captured) |= toDelta
			*p.colorOccupancy(undo.captured) |= toDelta
			// occDelta already has toDelta from the promo clear above;
			// adding the captured piece back cancels it out (XOR), so
			// we need to XOR again to cancel the cancel.
			occDelta ^= toDelta
		}

	default: // FlagNormal, FlagDoublePush
		piece := p.Board[to]
		p.Board[from] = piece
		p.Board[to] = undo.captured
		pieceBB := p.pieceBitboardFor(piece)
		pieceColorOcc := p.colorOccupancy(piece)
		delta := Bitboard(1)<<from | Bitboard(1)<<to
		*pieceBB ^= delta
		*pieceColorOcc ^= delta
		occDelta ^= delta
		if undo.captured != 0 {
			capDelta := Bitboard(1) << to
			*p.pieceBitboardFor(undo.captured) |= capDelta
			*p.colorOccupancy(undo.captured) |= capDelta
			occDelta ^= capDelta
		}
	}

	p.Occupied ^= occDelta
	p.Empty = ^p.Occupied

	// Restore cached king square if the king was the moving piece.
	movedPiece := p.Board[from]
	if movedPiece&types.TypeMask == types.King {
		if movedPiece.Color() == types.ColorWhite {
			p.KingSquares[0] = from
		} else {
			p.KingSquares[1] = from
		}
	}

	// Restore the pre-move en passant, castling, clock, eval, and hash state.
	p.EnPassantCapture = undo.enPassantCapture
	p.EnPassantTarget = undo.enPassantTarget
	p.CastlingRights = undo.castlingRights
	p.HalfmoveClock = undo.halfmoveClock
	p.EvalScore = undo.evalScore
	p.Hash = undo.hash

	// Fullmove number — decrement if we're undoing a black move (i.e., after
	// the side flip above, WhiteToMove is now false meaning black was to move).
	if !p.WhiteToMove {
		p.FullmoveNumber--
	}
}