package engine

import "webassemble/pkg/types"

type gameState struct {
	board            types.Board
	enPassantCapture int
	enPassantTarget  int
	whiteToMove      bool
	castlingRights   types.CastlingRights
}

var stateStack []gameState

func saveState() {
	stateStack = append(stateStack, gameState{
		board:            Board,
		enPassantCapture: enPassantCapture,
		enPassantTarget:  enPassantTarget,
		whiteToMove:       whiteToMove,
		castlingRights:    castlingRights,
	})
}

func restoreState() {
	n := len(stateStack) - 1
	saved := stateStack[n]
	stateStack = stateStack[:n]
	Board = saved.board
	enPassantCapture = saved.enPassantCapture
	enPassantTarget = saved.enPassantTarget
	whiteToMove = saved.whiteToMove
	castlingRights = saved.castlingRights
}

func promotionInt(p *types.Piece) int {
	if p == nil {
		return 0
	}
	return int(*p)
}

func GetValidMoves() []types.Move {
	pseudo := getPseudoLegalMoves()
	moverColor := types.ColorWhite
	if !whiteToMove {
		moverColor = types.ColorBlack
	}
	legal := make([]types.Move, 0, len(pseudo))
	for _, m := range pseudo {
		saveState()
		MakeMove(m.From, m.To, promotionInt(m.Promotion))
		if !IsInCheck(moverColor) {
			legal = append(legal, m)
		}
		restoreState()
	}
	return legal
}
