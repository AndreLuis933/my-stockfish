package ai

import "webassemble/pkg/engine"

// Evaluate returns the static evaluation from the perspective of the side to
// move. Positive = favorable for the side to move; negative = unfavorable.
//
// The score is maintained incrementally by Make/Unmake (material + piece-square
// tables), so this is an O(1) read — no board scan.
func Evaluate(p *engine.Position) int {
	if p.WhiteToMove {
		return p.EvalScore
	}
	return -p.EvalScore
}