//go:build js && wasm

package main

import (
	"encoding/json"
	"syscall/js"
	"webassemble/pkg/ai"
	"webassemble/pkg/engine"
	"webassemble/pkg/types"
)

func getBoard() interface{} {
	jsArray := js.Global().Get("Uint8Array").New(64)

	buf := make([]byte, 64)
	for i, p := range engine.Game.Board {
		buf[i] = byte(p)
	}

	js.CopyBytesToJS(jsArray, buf)
	return jsArray
}

func getValidMovesJS(_ js.Value, args []js.Value) interface{} {
	moves := engine.Game.LegalMovesSlice()

	data, err := json.Marshal(moves)
	if err != nil {
		return js.ValueOf(nil)
	}
	return js.ValueOf(string(data))
}

func getBoardJS(_ js.Value, _ []js.Value) interface{} {
	return getBoard()
}

func isCheckJS(_ js.Value, _ []js.Value) interface{} {
	return engine.KingCheck()
}

func gameStatusJS(_ js.Value, _ []js.Value) interface{} {
	return engine.CurrentStatus().String()
}

func makeMoveJS(_ js.Value, args []js.Value) interface{} {
	promotion := 0
	if len(args) > 2 && !args[2].IsUndefined() && args[2].Type() == js.TypeNumber {
		promotion = args[2].Int()
	}
	engine.MakeMove(args[0].Int(), args[1].Int(), promotion)
	return getBoard()
}

//rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1

func initBoardJs(_ js.Value, _ []js.Value) interface{} {
	engine.LoadFen(engine.StartingFEN)
	return getBoard()
}

// aiMoveJS runs a time-limited AI search and returns the best move as JSON.
func aiMoveJS(_ js.Value, args []js.Value) interface{} {
	timeLimitMs := 500
	if len(args) > 0 && !args[0].IsUndefined() && args[0].Type() == js.TypeNumber {
		timeLimitMs = args[0].Int()
	}
	result := ai.Search(engine.Game, timeLimitMs)
	return moveToJSON(result.Move)
}

// aiMoveDepthJS runs a fixed-depth AI search (no time limit) and returns the
// best move as JSON. Used for testing/benchmarking in the browser.
func aiMoveDepthJS(_ js.Value, args []js.Value) interface{} {
	depth := 4
	if len(args) > 0 && !args[0].IsUndefined() && args[0].Type() == js.TypeNumber {
		depth = args[0].Int()
	}
	result := ai.SearchFixedDepth(engine.Game, depth)
	return moveToJSON(result.Move)
}

func moveToJSON(move types.Move) interface{} {
	moveJSON := struct {
		From      int    `json:"from"`
		To        int    `json:"to"`
		Promotion *int   `json:"promotion,omitempty"`
	}{
		From: move.From,
		To:   move.To,
	}
	if move.Promotion != 0 {
		promo := int(move.Promotion)
		moveJSON.Promotion = &promo
	}
	data, err := json.Marshal(moveJSON)
	if err != nil {
		return js.ValueOf(nil)
	}
	return js.ValueOf(string(data))
}

func main() {
	e := js.Global().Get("Object").New()
	e.Set("validMovesChess", js.FuncOf(getValidMovesJS))
	e.Set("initBoard", js.FuncOf(initBoardJs))
	e.Set("makeMove", js.FuncOf(makeMoveJS))
	e.Set("isCheckJS", js.FuncOf(isCheckJS))
	e.Set("gameStatus", js.FuncOf(gameStatusJS))
	e.Set("aiMove", js.FuncOf(aiMoveJS))
	e.Set("aiMoveDepth", js.FuncOf(aiMoveDepthJS))
	js.Global().Set("goWasmEngine", e)
	select {}
}
