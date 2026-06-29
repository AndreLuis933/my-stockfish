package engine

import "webassemble/pkg/types"

type castleSide struct {
	color    types.Piece
	rights   types.CastlingRights
	kingFrom int
	kingTo   int
	rook     int
	empty    []int
	kingPath []int
	flag     types.MoveFlag
}

var castleSides = [4]castleSide{
	{types.ColorWhite, types.CastleWhiteK, 4, 6, 7, []int{5, 6}, []int{5, 6}, types.FlagCastleK},
	{types.ColorWhite, types.CastleWhiteQ, 4, 2, 0, []int{1, 2, 3}, []int{3, 2}, types.FlagCastleQ},
	{types.ColorBlack, types.CastleBlackK, 60, 62, 63, []int{61, 62}, []int{61, 62}, types.FlagCastleK},
	{types.ColorBlack, types.CastleBlackQ, 60, 58, 56, []int{57, 58, 59}, []int{59, 58}, types.FlagCastleQ},
}

func (p *Position) isPathAttacked(squares []int, byColor types.Piece) bool {
	for _, sq := range squares {
		if p.IsSquareAttacked(sq, byColor) {
			return true
		}
	}
	return false
}

func (p *Position) isEmpty(squares []int) bool {
	for _, idx := range squares {
		if p.Board[idx] != 0 {
			return false
		}
	}
	return true
}

func (p *Position) generateCastling(piece types.Piece, ml *MoveList) {
	enemy := oppositeColor(piece)
	for _, s := range castleSides {
		if s.color != piece.Color() { // only the side to move can castle
			continue
		}
		if p.CastlingRights&s.rights == 0 { // castling rights still present
			continue
		}
		if p.Board[s.kingFrom].TypePiece() != types.King || p.Board[s.kingFrom].Color() != piece.Color() { // king on its origin square
			continue
		}
		if p.Board[s.rook].TypePiece() != types.Rook || p.Board[s.rook].Color() != piece.Color() { // rook on its origin square
			continue
		}
		if !p.isEmpty(s.empty) { // squares between king and rook are empty
			continue
		}
		if p.IsSquareAttacked(s.kingFrom, enemy) { // king is not currently in check
			continue
		}
		if p.isPathAttacked(s.kingPath, enemy) { // king path (traverse + destination) not attacked
			continue
		}
		ml.Add(types.Move{From: uint8(s.kingFrom), To: uint8(s.kingTo), Flag: s.flag})
	}
}
