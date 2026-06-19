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

// Make applies a fully-populated move to this position.
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
//
// This version still uses snapshot/restore for legal-move filtering (see
// legal.go). Step 3 will add an Unmake method so search can avoid full copies.
func (p *Position) Make(move types.Move) {
	from, to := move.From, move.To
	piece := p.Board[from]

	// Save en passant state before clearing it — the FlagEnPassant case
	// needs the old EnPassantCapture square to remove the captured pawn.
	prevEnPassantCapture := p.EnPassantCapture
	p.EnPassantCapture, p.EnPassantTarget = -1, -1

	switch move.Flag {
	case types.FlagEnPassant:
		p.Board[prevEnPassantCapture] = 0
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
			// Fallback: if a promotion move was built without a Promotion
			// piece (e.g. from the raw bridge path), default to Queen.
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