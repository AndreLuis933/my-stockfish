package book

import (
	"webassemble/pkg/engine"
	"webassemble/pkg/types"
)

// DecodePolyglotMove decodes a 16-bit Polyglot move into an engine Move.
//
// Polyglot move encoding (big-endian uint16):
//
//	bits  0-5: destination square (LERF: a1=0, b1=1, ... h8=63)
//	bits 6-11: origin square
//	bits 12-14: promotion (0=none, 1=Knight, 2=Bishop, 3=Rook, 4=Queen)
//	bit 15: unused
//
// The returned Move has From/To/Promotion set. Flag and Captured are left
// zero — the caller should match (From, To) against LegalMoves to resolve
// flags correctly (castling, en passant, promotion flags).
func DecodePolyglotMove(polyMove uint16) types.Move {
	to := uint8(polyMove & 0x3F)
	from := uint8((polyMove >> 6) & 0x3F)
	promoPart := (polyMove >> 12) & 0x7

	move := types.Move{
		From: from,
		To:   to,
	}

	if promoPart > 0 {
		var promoType types.Piece
		switch promoPart {
		case 1:
			promoType = types.Knight
		case 2:
			promoType = types.Bishop
		case 3:
			promoType = types.Rook
		case 4:
			promoType = types.Queen
		}
		// Set promotion piece with the correct color bits.
		// We need to determine the color from the from-square's piece
		// in the position, but since DecodePolyglotMove doesn't have
		// access to the board, the caller must set the color bits.
		// For now, set the type bits only; the caller (MatchLegalMove)
		// will replace this with the correct legal Move from LegalMoves.
		move.Promotion = promoType
	}

	return move
}

// MatchLegalMove finds the legal move in p that matches the book move's
// (from, to) squares. This resolves Flag, Captured, and Promotion correctly
// by searching the legal move list. Returns the matched move and true, or
// a zero Move and false if no legal move matches.
//
// This is the correct way to apply a book move: the polyglot encoding
// doesn't carry flag information, so we match against legal moves to get
// the full Move struct that Make() expects.
func MatchLegalMove(p *engine.Position, bookMove types.Move) (types.Move, bool) {
	var ml engine.MoveList
	p.LegalMoves(&ml)

	for i := 0; i < ml.Len(); i++ {
		m := ml.Get(i)
		if m.From == bookMove.From && m.To == bookMove.To {
			// For promotions, also check the promotion type matches.
			// The book move's Promotion only has type bits (no color),
			// so compare against the type bits of the legal move's promotion.
			if bookMove.Promotion != 0 {
				if m.Promotion.TypePiece() != bookMove.Promotion.TypePiece() {
					continue
				}
			}
			return m, true
		}
	}

	return types.Move{}, false
}