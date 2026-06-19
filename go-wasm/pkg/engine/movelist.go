package engine

import "webassemble/pkg/types"

// MoveList is a fixed-capacity move buffer. It lives on the stack when the
// caller declares it as a local var (zero allocations) and is written to
// directly by the move generators via a pointer.
//
// Capacity 256 covers the theoretical maximum of 218 legal moves in any
// reachable chess position, with headroom.
type MoveList struct {
	moves [256]types.Move
	n     int
}

func (ml *MoveList) Add(m types.Move) {
	ml.moves[ml.n] = m
	ml.n++
}

func (ml *MoveList) Len() int { return ml.n }

func (ml *MoveList) Get(i int) types.Move { return ml.moves[i] }

func (ml *MoveList) Clear() { ml.n = 0 }

// Slice returns a view over the valid moves. The returned slice aliases the
// MoveList's backing array — if the MoveList is stack-allocated and this slice
// escapes, the whole MoveList moves to the heap.
func (ml *MoveList) Slice() []types.Move { return ml.moves[:ml.n] }