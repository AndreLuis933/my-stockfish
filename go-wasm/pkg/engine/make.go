package engine

import "webassemble/pkg/types"

// MakeMove applies a move to the global Game position.
// Kept as a free function for the WASM bridge; delegates to the method.
func MakeMove(from, to, promotion int) {
	Game.MakeMove(from, to, promotion)
}

// MakeMove applies a move to this position.
//
// It handles: en passant capture (removes the pawn behind the target square),
// pawn promotion (replaces the pawn with the chosen piece), castling (moves
// the rook alongside the king), en passant target setup after a double push,
// and castling-rights updates when king/rook moves or a rook is captured.
//
// This is the "snapshot/restore" version — it does NOT undo moves. The legal
// move filter and Perft save a full copy of the position before calling this,
// then restore. A later refactor (Step 3 of the plan) will add an Unmake method
// that reverses a move incrementally, which is much faster for AI search.
func (p *Position) MakeMove(from, to, promotion int) {
	piece := p.Board[from]
	pieceTo := p.Board[to]

	// En passant capture: the captured pawn is NOT on `to`, it's on the
	// square we stored in EnPassantCapture when the double push happened.
	if piece&types.Pawn == types.Pawn && to == p.EnPassantTarget && p.EnPassantCapture != -1 {
		p.Board[p.EnPassantCapture] = 0
	}

	// Move the piece (or place the promoted piece).
	p.Board[from] = 0
	if promotion != 0 {
		p.Board[to] = types.Piece(promotion)
	} else {
		p.Board[to] = piece
	}

	p.WhiteToMove = !p.WhiteToMove

	// Reset en passant state — it only lasts one move.
	p.EnPassantCapture, p.EnPassantTarget = -1, -1

	// If this was a double pawn push, set the new en passant target.
	if piece&types.Pawn == types.Pawn {
		diff := to - from
		if diff == 2*boardSize || diff == -2*boardSize {
			p.EnPassantCapture = to
			p.EnPassantTarget = (from + to) / 2
		}
	}

	// Castling: if the king moved two squares, move the rook to the other
	// side of the king. Also clear castling rights for that color.
	if piece&types.King == types.King {
		if piece.Color() == types.ColorWhite {
			p.CastlingRights &^= types.CastleWhiteAll
		} else {
			p.CastlingRights &^= types.CastleBlackAll
		}
		dif := to - from
		switch dif {
		case 2: // kingside: rook h->f
			rook := p.Board[to+1]
			p.Board[to+1] = 0
			p.Board[to-1] = rook
		case -2: // queenside: rook a->d
			rook := p.Board[to-2]
			p.Board[to-2] = 0
			p.Board[to+1] = rook
		}
	}

	// Castling rights are lost when a rook moves from its origin corner,
	// or when a rook is captured on a corner square.
	if piece&types.Rook == types.Rook || pieceTo&types.Rook == types.Rook {
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
}