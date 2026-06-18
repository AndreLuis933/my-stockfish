package main

import (
	"webassemble/pkg/engine"
)

func main(){
	engine.LoadFen("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
	engine.GetValidMoves()
}