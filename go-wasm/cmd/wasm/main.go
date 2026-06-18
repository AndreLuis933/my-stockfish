//go:build js && wasm

package main

import (
	"encoding/json"
	"syscall/js"
	"webassemble/pkg/engine"
)

func getBoard() interface{} {
	jsArray := js.Global().Get("Uint8Array").New(64)

	buf := make([]byte, 64)
	for i, p := range engine.Board {
		buf[i] = byte(p)
	}

	js.CopyBytesToJS(jsArray, buf)
	return jsArray
}

func getValidMovesJS(_ js.Value, args []js.Value) interface{} {
	moves := engine.GetValidMoves() // []Move

	data, err := json.Marshal(moves)
	if err != nil {
		return js.ValueOf(nil)
	}
	return js.ValueOf(string(data))
}

func getBoardJS(_ js.Value, _ []js.Value) interface{} {
	return getBoard()
}

func makeMoveJS(_ js.Value, args []js.Value) interface{} {
	engine.MakeMovement(args[0].Int(), args[1].Int())
	return getBoard()
}
//rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1

func initBoardJs(_ js.Value, _ []js.Value) interface{} {
	engine.LoadFen("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
	return getBoard()
}

func main() {
	e := js.Global().Get("Object").New()
	e.Set("validMovesChess", js.FuncOf(getValidMovesJS))
	e.Set("initBoard", js.FuncOf(initBoardJs))
	e.Set("makeMove", js.FuncOf(makeMoveJS))
	js.Global().Set("goWasmEngine", e)
	select {}
}
