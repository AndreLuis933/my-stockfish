package main

import (
	"fmt"

	"webassemble/pkg/engine"
)

func main() {
	engine.LoadFen("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
	moves := engine.GetValidMoves()
	fmt.Printf("moves: %d\n", len(moves))
	fmt.Printf("status: %s\n", engine.CurrentStatus().String())
	fmt.Printf("check: %d\n", engine.KingCheck())
}