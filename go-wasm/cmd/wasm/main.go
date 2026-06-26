//go:build js && wasm

package main

import (
	"encoding/json"
	"strings"
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

// sharedTT is the persistent transposition table, reused across searches
// within a game. Entries accumulate across moves (improving hit rates for
// recurring positions) and are cleared on initBoard (new game). This mirrors
// the UCI engine's session-level TT.
var sharedTT = engine.DefaultTranspositionTable()

func initBoardJs(_ js.Value, _ []js.Value) interface{} {
	sharedTT.Clear()
	engine.LoadFen(engine.StartingFEN)
	return getBoard()
}

// aiMoveJS runs a time-limited AI search and returns the best move as JSON.
func aiMoveJS(_ js.Value, args []js.Value) interface{} {
	timeLimitMs := 500
	if len(args) > 0 && !args[0].IsUndefined() && args[0].Type() == js.TypeNumber {
		timeLimitMs = args[0].Int()
	}
	result := ai.SearchWithTT(engine.Game, timeLimitMs, nil, sharedTT)
	return moveToJSON(result.Move)
}

// aiMoveDepthJS runs a fixed-depth AI search (no time limit) and returns the
// best move as JSON. Used for testing/benchmarking in the browser.
func aiMoveDepthJS(_ js.Value, args []js.Value) interface{} {
	depth := 4
	if len(args) > 0 && !args[0].IsUndefined() && args[0].Type() == js.TypeNumber {
		depth = args[0].Int()
	}
	result := ai.SearchFixedDepthWithTT(engine.Game, depth, nil, sharedTT)
	return moveToJSON(result.Move)
}

// aiAnalysisJS runs a time-limited AI search and returns the best move plus
// the evaluation score and search depth as JSON. Used by the "Analisar" button
// to show the engine's assessment of the current position.
func aiAnalysisJS(_ js.Value, args []js.Value) interface{} {
	timeLimitMs := 1000
	if len(args) > 0 && !args[0].IsUndefined() && args[0].Type() == js.TypeNumber {
		timeLimitMs = args[0].Int()
	}
	result := ai.SearchWithTT(engine.Game, timeLimitMs, nil, sharedTT)
	return analysisToJSON(result)
}

func analysisToJSON(result ai.SearchResult) interface{} {
	data := struct {
		From      int    `json:"from"`
		To        int    `json:"to"`
		Promotion *int   `json:"promotion,omitempty"`
		Score     int    `json:"score"`
		Depth     int    `json:"depth"`
		Nodes     int    `json:"nodes"`
		TimeMs    int64  `json:"timeMs"`
	}{
		From:   result.Move.From,
		To:     result.Move.To,
		Score:  result.Score,
		Depth:  result.Depth,
		Nodes:  result.Nodes,
		TimeMs: result.TimeMs,
	}
	if result.Move.Promotion != 0 {
		promo := int(result.Move.Promotion)
		data.Promotion = &promo
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return js.ValueOf(nil)
	}
	return js.ValueOf(string(raw))
}

// pvMoveJSON is the JSON shape for a single move in a Multi-PV line. Matches
// the frontend MoveData contract: {from, to, promotion?}.
type pvMoveJSON struct {
	From      int  `json:"from"`
	To        int  `json:"to"`
	Promotion *int `json:"promotion,omitempty"`
}

// multiPvLineJSON is one line of a Multi-PV result.
type multiPvLineJSON struct {
	Moves  []pvMoveJSON `json:"moves"`
	Score  int         `json:"score"`
	Depth  int         `json:"depth"`
	Nodes  int         `json:"nodes"`
	TimeMs int64       `json:"timeMs"`
}

// aiMultiPvJS runs a time-limited Multi-PV search and returns numLines
// principal variations as JSON. Used by the analysis panel to show multiple
// candidate lines ranked by eval.
func aiMultiPvJS(_ js.Value, args []js.Value) interface{} {
	timeLimitMs := 1000
	if len(args) > 0 && !args[0].IsUndefined() && args[0].Type() == js.TypeNumber {
		timeLimitMs = args[0].Int()
	}
	numLines := 3
	if len(args) > 1 && !args[1].IsUndefined() && args[1].Type() == js.TypeNumber {
		numLines = args[1].Int()
	}
	lines := ai.SearchMultiPV(engine.Game, timeLimitMs, numLines, nil, sharedTT)

	out := make([]multiPvLineJSON, 0, len(lines))
	for _, line := range lines {
		moves := make([]pvMoveJSON, 0, len(line.Moves))
		for _, m := range line.Moves {
			mv := pvMoveJSON{From: m.From, To: m.To}
			if m.Promotion != 0 {
				promo := int(m.Promotion)
				mv.Promotion = &promo
			}
			moves = append(moves, mv)
		}
		out = append(out, multiPvLineJSON{
			Moves:  moves,
			Score:  line.Score,
			Depth:  line.Depth,
			Nodes:  line.Nodes,
			TimeMs: line.TimeMs,
		})
	}

	raw, err := json.Marshal(out)
	if err != nil {
		return js.ValueOf("[]")
	}
	return js.ValueOf(string(raw))
}

// fenJS returns the FEN string of the current position. Used by the bottom bar
// to display the live FEN field.
func fenJS(_ js.Value, _ []js.Value) interface{} {
	return js.ValueOf(engine.Game.FEN())
}

// sanJS generates SAN for a move (from, to, optional promotion) in the
// current position. The position must be BEFORE the move is made.
func sanJS(_ js.Value, args []js.Value) interface{} {
	from := args[0].Int()
	to := args[1].Int()
	promotion := 0
	if len(args) > 2 && !args[2].IsUndefined() && args[2].Type() == js.TypeNumber {
		promotion = args[2].Int()
	}

	piece := engine.Game.Board[from]
	move := types.Move{From: from, To: to}

	// Infer the flag (same logic as MakeMove)
	switch {
	case piece&types.Pawn != 0 && promotion != 0:
		move.Flag = types.FlagPromotion
		move.Promotion = types.Piece(promotion)
	case piece&types.King != 0 && abs(to-from) == 2:
		if to > from {
			move.Flag = types.FlagCastleK
		} else {
			move.Flag = types.FlagCastleQ
		}
	case piece&types.Pawn != 0 && abs(to-from) == 2*8:
		move.Flag = types.FlagDoublePush
	case piece&types.Pawn != 0 && to == engine.Game.EnPassantTarget && engine.Game.EnPassantCapture != -1:
		move.Flag = types.FlagEnPassant
		move.Captured = engine.Game.Board[engine.Game.EnPassantCapture]
	default:
		if engine.Game.Board[to] != 0 {
			move.Captured = engine.Game.Board[to]
		}
	}

	san, err := engine.Game.ToSan(move)
	if err != nil {
		return js.ValueOf("")
	}
	return js.ValueOf(san)
}

// applyPgnJS replays a PGN's SAN moves on the engine, returning a JSON array
// of history entries (san, from, to, boardBefore, boardAfter, checkSquare, status).
// This replaces the frontend's N-round-trip loadPgn loop with a single call.
func applyPgnJS(_ js.Value, args []js.Value) interface{} {
	pgn := args[0].String()

	engine.LoadFen(engine.StartingFEN)

	sanMoves := parsePgnMoves(pgn)
	if len(sanMoves) == 0 {
		return js.ValueOf("[]")
	}

	type histEntry struct {
		San         string `json:"san"`
		From        int    `json:"from"`
		To          int    `json:"to"`
		Promotion   *int   `json:"promotion,omitempty"`
		BoardBefore []int  `json:"boardBefore"`
		BoardAfter  []int  `json:"boardAfter"`
		CheckSquare int    `json:"checkSquare"`
		IsCheckmate bool   `json:"isCheckmate"`
	}

	entries := make([]histEntry, 0, len(sanMoves))

	for _, san := range sanMoves {
		move, err := engine.Game.SanToMove(san)
		if err != nil {
			return js.ValueOf("[]")
		}

		boardBefore := boardToSlice(engine.Game.Board)

		engine.Game.Make(move)

		checkSq := engine.KingCheck()
		status := engine.Game.CurrentStatus()
		isCheckmate := status == engine.StatusWhiteWins || status == engine.StatusBlackWins

		boardAfter := boardToSlice(engine.Game.Board)

		entry := histEntry{
			San:         san,
			From:        move.From,
			To:          move.To,
			BoardBefore: boardBefore,
			BoardAfter:  boardAfter,
			CheckSquare: checkSq,
			IsCheckmate: isCheckmate,
		}
		if move.Promotion != 0 {
			p := int(move.Promotion)
			entry.Promotion = &p
		}
		entries = append(entries, entry)
	}

	raw, err := json.Marshal(entries)
	if err != nil {
		return js.ValueOf("[]")
	}
	return js.ValueOf(string(raw))
}

func boardToSlice(board [64]types.Piece) []int {
	out := make([]int, 64)
	for i, p := range board {
		out[i] = int(p)
	}
	return out
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// parsePgnMoves extracts SAN move tokens from a PGN string (strips headers,
// comments, variations, result tokens).
func parsePgnMoves(pgn string) []string {
	var moves []string
	for _, line := range strings.Split(pgn, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "[") {
			continue
		}
		for _, tok := range strings.Fields(line) {
			tok = strings.TrimPrefix(tok, "1.")
			tok = strings.TrimPrefix(tok, "2.")
			tok = strings.TrimPrefix(tok, "3.")
			tok = strings.TrimPrefix(tok, "4.")
			tok = strings.TrimPrefix(tok, "5.")
			tok = strings.TrimPrefix(tok, "6.")
			tok = strings.TrimPrefix(tok, "7.")
			tok = strings.TrimPrefix(tok, "8.")
			tok = strings.TrimPrefix(tok, "9.")
			if tok == "" || tok == "1-0" || tok == "0-1" || tok == "1/2-1/2" || tok == "*" {
				continue
			}
			moves = append(moves, tok)
		}
	}
	return moves
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
	e.Set("aiAnalysis", js.FuncOf(aiAnalysisJS))
	e.Set("aiMultiPv", js.FuncOf(aiMultiPvJS))
	e.Set("fen", js.FuncOf(fenJS))
	e.Set("san", js.FuncOf(sanJS))
	e.Set("applyPgn", js.FuncOf(applyPgnJS))
	js.Global().Set("goWasmEngine", e)
	select {}
}
