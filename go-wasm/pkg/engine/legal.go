package engine

import "webassemble/pkg/types"

var saved struct {
    board             types.Board
    enPassantCapture  int
    enPassantTarget   int
    whiteToMove       bool
    castlingRights   types.CastlingRights
}

func saveState() {
    saved.board            = Board
    saved.enPassantCapture = enPassantCapture
    saved.enPassantTarget  = enPassantTarget
    saved.whiteToMove      = whiteToMove
    saved.castlingRights   = castlingRights
}

func restoreState() {
    Board            = saved.board
    enPassantCapture = saved.enPassantCapture
    enPassantTarget  = saved.enPassantTarget
    whiteToMove      = saved.whiteToMove
    castlingRights   = saved.castlingRights
}

func promotionInt(p *types.Piece) int {
    if p == nil { return 0 }
    return int(*p)
}


func GetValidMoves() []types.Move {
    pseudo := getPseudoLegalMoves()
    moverColor := types.ColorWhite
    if !whiteToMove { moverColor = types.ColorBlack }
    legal := make([]types.Move, 0, len(pseudo))
    for _, m := range pseudo {
        saveState()
        MakeMovement(m.From, m.To, promotionInt(m.Promotion))
        if !IsInCheck(moverColor) {
            legal = append(legal, m)
        }
        restoreState()
    }
    return legal
}
