package ai

import (
	"webassemble/pkg/engine"
	"webassemble/pkg/types"
)

// principalVariation reconstructs the PV from the transposition table starting
// at the current position. It follows the TT move at each ply until no entry is
// found, the entry has no move, a cycle is detected, or maxLen plies are reached.
//
// The position is restored to its original state via Unmake on every path, so
// the caller's p is unchanged when this returns.
//
// TT entries store moves via PackMove, which loses the MoveFlag and Captured
// fields (only from/to/promotion survive). Before calling Make, we reconstruct
// the flag and captured piece from the board so Make/Unmake stay balanced:
//   - King moving two files → castling (K or Q side by direction)
//   - Pawn moving diagonally to an empty square → en passant
//   - Pawn reaching the last rank → promotion (UnpackMove already sets the flag)
//   - Otherwise → normal move; Captured = board[to] before Make
func principalVariation(p *engine.Position, tt *engine.TranspositionTable, maxLen int) []types.Move {
	if tt == nil || maxLen <= 0 {
		return nil
	}

	var pv []types.Move
	visited := make(map[uint64]struct{}, maxLen)

	for len(pv) < maxLen {
		hash := p.Hash
		if _, seen := visited[hash]; seen {
			break
		}
		visited[hash] = struct{}{}

		entry, ok := tt.Probe(hash)
		if !ok {
			break
		}
		raw, hasMove := engine.UnpackMove(entry.Move)
		if !hasMove {
			break
		}

		move := reconstructMove(p, raw)
		moverColor := sideColor(p)
		p.Make(move)
		if p.IsInCheck(moverColor) {
			p.Unmake(move)
			break
		}
		pv = append(pv, move)
	}

	for i := len(pv) - 1; i >= 0; i-- {
		p.Unmake(pv[i])
	}

	return pv
}

// reconstructMove fills in the Flag and Captured fields of a TT-unpacked move
// by inspecting the board. PackMove/UnpackMove only preserve from/to/promotion,
// so Make needs the full move to apply and unmake correctly.
func reconstructMove(p *engine.Position, raw types.Move) types.Move {
	piece := p.Board[raw.From]
	captured := p.Board[raw.To]
	flag := types.FlagNormal

	pieceType := piece & types.TypeMask
	fromRow, fromCol := int(raw.From)/8, int(raw.From)%8
	toRow, toCol := int(raw.To)/8, int(raw.To)%8

	switch pieceType {
	case types.King:
		if pvAbs(fromCol-toCol) == 2 {
			if toCol == 6 {
				flag = types.FlagCastleK
			} else {
				flag = types.FlagCastleQ
			}
		}
	case types.Pawn:
		if raw.Promotion != 0 {
			flag = types.FlagPromotion
		} else if fromRow == toRow-2 || fromRow == toRow+2 {
			flag = types.FlagDoublePush
		} else if fromCol != toCol && captured == 0 {
			flag = types.FlagEnPassant
			if p.EnPassantCapture >= 0 && int(raw.To) == p.EnPassantTarget {
				if piece.IsWhite() {
					captured = p.Board[int(raw.To)+8]
				} else {
					captured = p.Board[int(raw.To)-8]
				}
			}
		}
	}

	return types.Move{
		From:      raw.From,
		To:        raw.To,
		Promotion: raw.Promotion,
		Flag:      flag,
		Captured:  captured,
	}
}

func pvAbs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}