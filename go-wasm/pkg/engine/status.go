package engine

import "webassemble/pkg/types"

type GameStatus int

const (
	StatusPlaying    GameStatus = 0
	StatusWhiteWins  GameStatus = 1
	StatusBlackWins  GameStatus = 2
	StatusDraw       GameStatus = 3
)

var statusNames = map[GameStatus]string{
	StatusPlaying:   "playing",
	StatusWhiteWins: "white-wins",
	StatusBlackWins: "black-wins",
	StatusDraw:      "draw",
}

func (s GameStatus) String() string {
	return statusNames[s]
}

func (s GameStatus) IsGameOver() bool {
	return s != StatusPlaying
}

// gameStatusFor inspects the position for the side that has the move.
// If that side has no legal moves: in check → checkmate (other side wins),
// not in check → stalemate (draw). Otherwise the game is still going.
func gameStatusFor(sideToMoveColor types.Piece, moves []types.Move, inCheck bool) GameStatus {
	if len(moves) > 0 {
		return StatusPlaying
	}
	if inCheck {
		if sideToMoveColor == types.ColorWhite {
			return StatusBlackWins
		}
		return StatusWhiteWins
	}
	return StatusDraw
}

// CurrentStatus computes the current game status from the live engine state.
// Used by the frontend after a move and by the AI to know when to stop searching.
func CurrentStatus() GameStatus {
	color := isWhite(whiteToMove)
	moves := GetValidMoves()
	inCheck := IsInCheck(color)
	return gameStatusFor(color, moves, inCheck)
}