package engine

import "webassemble/pkg/types"

type GameStatus int

const (
	StatusPlaying   GameStatus = 0
	StatusWhiteWins GameStatus = 1
	StatusBlackWins GameStatus = 2
	StatusDraw      GameStatus = 3
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

// statusFor inspects the position for the side that has the move.
// If that side has no legal moves: in check → checkmate (other side wins),
// not in check → stalemate (draw). Otherwise the game is still going.
func statusFor(sideToMoveColor types.Piece, moveCount int, inCheck bool) GameStatus {
	if moveCount > 0 {
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

// CurrentStatus computes the current game status from the live Game position.
// Used by the frontend after a move and by the AI to know when to stop searching.
func CurrentStatus() GameStatus {
	return Game.CurrentStatus()
}

// CurrentStatus computes the status for this position. Checks for
// checkmate/stalemate first (no legal moves), then draw rules (50-move,
// threefold repetition, insufficient material). A checkmated position is
// never a draw, even if the 50-move clock is high or the position repeated.
func (p *Position) CurrentStatus() GameStatus {
	color := p.colorOfSide()
	var ml MoveList
	p.LegalMoves(&ml)
	inCheck := p.IsInCheck(color)
	if ml.n == 0 {
		return statusFor(color, ml.n, inCheck)
	}
	if p.IsDraw() {
		return StatusDraw
	}
	return statusFor(color, ml.n, inCheck)
}