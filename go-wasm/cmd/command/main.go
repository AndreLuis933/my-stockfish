package main

import (
	"fmt"

	"webassemble/pkg/engine"
)

func main() {
	fen := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	for i := 1; i <= 6; i++ {
		engine.LoadFen(fen)
		nodes := engine.Perft(i)
		fmt.Printf("depth %d  nodes %d\n", i, nodes)
	}
}
