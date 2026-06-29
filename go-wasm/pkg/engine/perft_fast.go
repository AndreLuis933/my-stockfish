package engine

import "webassemble/pkg/types"

// fastUndo holds the minimal state needed to reverse a fastMake call.
// Used only by the Perft fast path; kept on the recursion stack.
type fastUndo struct {
	captured         types.Piece
	captureSquare    int
	enPassantCapture int
	enPassantTarget  int
	castlingRights   types.CastlingRights
	kingSquares      [2]int
	whiteToMove      bool
}

// PerftFast counts Perft nodes using a lightweight make/unmake that skips
// hash, eval, undo stack, and fullmove/halfmove clock updates.
//
// It produces identical counts to Perft and is the right benchmark for raw
// move-generation + minimal state-maintenance throughput.
func (p *Position) PerftFast(depth int) int {
	if depth == 0 {
		return 1
	}

	var ml MoveList
	p.PseudoLegalMoves(&ml)
	moverColor := p.colorOfSide()
	nodes := 0
	for i := 0; i < ml.n; i++ {
		undo := p.fastMake(ml.moves[i])
		if !p.IsInCheck(moverColor) {
			nodes += p.PerftFast(depth - 1)
		}
		p.fastUnmake(ml.moves[i], undo)
	}
	return nodes
}

// fastMake applies a fully-populated move and returns a snapshot sufficient
// for fastUnmake to restore the position.
func (p *Position) fastMake(move types.Move) fastUndo {
	from, to := int(move.From), int(move.To)
	piece := p.Board[from]

	captureSquare := to
	if move.Flag == types.FlagEnPassant {
		captureSquare = p.EnPassantCapture
	}

	undo := fastUndo{
		captured:         move.Captured,
		captureSquare:    captureSquare,
		enPassantCapture: p.EnPassantCapture,
		enPassantTarget:  p.EnPassantTarget,
		castlingRights:   p.CastlingRights,
		kingSquares:      p.KingSquares,
		whiteToMove:      p.WhiteToMove,
	}

	p.EnPassantCapture, p.EnPassantTarget = -1, -1

	isWhite := piece.Color() == types.ColorWhite
	var colorOcc *Bitboard
	if isWhite {
		colorOcc = &p.WhitePieces
	} else {
		colorOcc = &p.BlackPieces
	}

	var occDelta Bitboard

	switch move.Flag {
	case types.FlagEnPassant:
		p.Board[captureSquare] = 0
		p.Board[from] = 0
		p.Board[to] = piece

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

	case types.FlagPromotion:
		p.Board[from] = 0
		promoPiece := move.Promotion
		if promoPiece == 0 {
			promoPiece = piece | types.Queen
		}
		p.Board[to] = promoPiece

		fromDelta := Bitboard(1) << from
		*p.pieceBitboardFor(piece) &^= fromDelta
		*colorOcc &^= fromDelta
		occDelta ^= fromDelta

		toDelta := Bitboard(1) << to
		*p.pieceBitboardFor(promoPiece) |= toDelta
		*colorOcc |= toDelta
		occDelta ^= toDelta

		if move.Captured != 0 {
			capColorOcc := p.colorOccupancy(move.Captured)
			*p.pieceBitboardFor(move.Captured) &^= toDelta
			*capColorOcc &^= toDelta
			occDelta ^= toDelta
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
			*p.pieceBitboardFor(move.Captured) &^= Bitboard(1) << to
			*capColorOcc &^= Bitboard(1) << to
			occDelta ^= Bitboard(1) << to
		}
	}

	p.Occupied ^= occDelta
	p.Empty = ^p.Occupied

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

	p.WhiteToMove = !p.WhiteToMove
	return undo
}

// fastUnmake reverses a fastMake call using the snapshot returned by fastMake.
func (p *Position) fastUnmake(move types.Move, undo fastUndo) {
	from, to := int(move.From), int(move.To)

	p.WhiteToMove = !p.WhiteToMove

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

		toDelta := Bitboard(1) << to
		*p.pieceBitboardFor(promoPiece) &^= toDelta
		*p.colorOccupancy(promoPiece) &^= toDelta
		occDelta ^= toDelta

		fromDelta := Bitboard(1) << from
		*p.pieceBitboardFor(pawn) |= fromDelta
		*p.colorOccupancy(pawn) |= fromDelta
		occDelta ^= fromDelta

		if undo.captured != 0 {
			*p.pieceBitboardFor(undo.captured) |= toDelta
			*p.colorOccupancy(undo.captured) |= toDelta
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

	movedPiece := p.Board[from]
	if movedPiece&types.TypeMask == types.King {
		if movedPiece.Color() == types.ColorWhite {
			p.KingSquares[0] = from
		} else {
			p.KingSquares[1] = from
		}
	}

	p.EnPassantCapture = undo.enPassantCapture
	p.EnPassantTarget = undo.enPassantTarget
	p.CastlingRights = undo.castlingRights
}
