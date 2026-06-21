package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"webassemble/pkg/ai"
	"webassemble/pkg/engine"
	"webassemble/pkg/types"
)

const (
	engineName   = "my-stockfish"
	engineAuthor = "Andre"
)

// uciSession holds the state of a UCI conversation: the current position
// and the search goroutine lifecycle. The position is owned by the main
// goroutine (reading stdin); the search runs in its own goroutine and
// communicates only via channels.
type uciSession struct {
	pos engine.Position
	tt  *engine.TranspositionTable

	searchMu   sync.Mutex
	stopCh     chan struct{}
	searchDone chan struct{}
}

func newSession() *uciSession {
	s := &uciSession{
		tt: engine.DefaultTranspositionTable(),
	}
	s.pos.LoadFen(engine.StartingFEN)
	return s
}

// ── UCI move string conversion ─────────────────────────────────────

// squareToUCI converts a 0-63 board index to algebraic notation (e.g. "e4").
// The engine's board layout is rank-major with index 0 = a1, index 63 = h8:
// index = rank*8 + file, where rank 0 = rank 1, file 0 = file a.
func squareToUCI(idx int) string {
	file := idx % 8
	rank := idx / 8
	return string(rune('a'+file)) + strconv.Itoa(rank+1)
}

// uciToSquare converts algebraic notation (e.g. "e4") to a 0-63 index.
// Returns -1 on invalid input. Index = rank*8 + file (0 = a1).
func uciToSquare(s string) int {
	if len(s) != 2 {
		return -1
	}
	file := int(s[0] - 'a')
	rank := int(s[1] - '1') // 0-indexed: rank 1 → 0, rank 8 → 7
	if file < 0 || file > 7 || rank < 0 || rank > 7 {
		return -1
	}
	return rank*8 + file
}

// moveToUCI serializes an engine Move to UCI format: from+to plus an
// optional promotion letter (q/r/b/n). Example: "e2e4", "e7e8q".
func moveToUCI(m types.Move) string {
	s := squareToUCI(m.From) + squareToUCI(m.To)
	if m.Promotion != 0 {
		switch m.Promotion & types.TypeMask {
		case types.Queen:
			s += "q"
		case types.Rook:
			s += "r"
		case types.Bishop:
			s += "b"
		case types.Knight:
			s += "n"
		}
	}
	return s
}

// parseUCIMove parses a UCI move string (e.g. "e2e4", "e7e8q") against the
// legal moves of the current position and returns the matching Move.
// Returns ok=false if the string is not a legal move.
func (s *uciSession) parseUCIMove(uci string) (types.Move, bool) {
	if len(uci) < 4 || len(uci) > 5 {
		return types.Move{}, false
	}
	from := uciToSquare(uci[0:2])
	to := uciToSquare(uci[2:4])
	if from == -1 || to == -1 {
		return types.Move{}, false
	}

	var promoLetter byte
	if len(uci) == 5 {
		promoLetter = uci[4]
	}

	var legal engine.MoveList
	s.pos.LegalMoves(&legal)
	for i := 0; i < legal.Len(); i++ {
		m := legal.Get(i)
		if m.From != from || m.To != to {
			continue
		}
		if promoLetter == 0 {
			if m.Promotion == 0 {
				return m, true
			}
			continue
		}
		if m.Promotion == 0 {
			continue
		}
		var expected types.Piece
		switch promoLetter {
		case 'q':
			expected = types.Queen
		case 'r':
			expected = types.Rook
		case 'b':
			expected = types.Bishop
		case 'n':
			expected = types.Knight
		default:
			return types.Move{}, false
		}
		if m.Promotion&types.TypeMask == expected {
			return m, true
		}
	}
	return types.Move{}, false
}

// applyUCIMove parses and applies a UCI move string to the current position.
// Returns ok=false (and leaves the position unchanged) if the move is illegal.
func (s *uciSession) applyUCIMove(uci string) bool {
	m, ok := s.parseUCIMove(uci)
	if !ok {
		return false
	}
	s.pos.Make(m)
	return true
}

// ── Command handlers ───────────────────────────────────────────────

func (s *uciSession) handleUci() {
	fmt.Println("id name", engineName)
	fmt.Println("id author", engineAuthor)
	fmt.Println("uciok")
}

func (s *uciSession) handleIsready() {
	fmt.Println("readyok")
}

func (s *uciSession) handleUcinewgame() {
	s.stopSearch()
	s.pos.LoadFen(engine.StartingFEN)
	s.tt.Clear()
}

// handlePosition parses "position [startpos | fen <FEN>] [moves ...]".
// It stops any running search first (a position change mid-search is
// undefined behavior under UCI; we make it well-defined by aborting).
func (s *uciSession) handlePosition(parts []string) {
	s.stopSearch()

	i := 0
	if len(parts) == 0 {
		s.pos.LoadFen(engine.StartingFEN)
		return
	}

	switch parts[0] {
	case "startpos":
		s.pos.LoadFen(engine.StartingFEN)
		i = 1
	case "fen":
		// FEN is 6 space-separated fields; collect them.
		fenFields := []string{}
		i = 1
		for i < len(parts) && len(fenFields) < 6 {
			fenFields = append(fenFields, parts[i])
			i++
		}
		if len(fenFields) < 4 {
			fmt.Println("info string invalid fen")
			return
		}
		s.pos.LoadFen(strings.Join(fenFields, " "))
	default:
		fmt.Println("info string unknown position token:", parts[0])
		return
	}

	// Optional "moves m1 m2 ..."
	if i < len(parts) && parts[i] == "moves" {
		for j := i + 1; j < len(parts); j++ {
			if !s.applyUCIMove(parts[j]) {
				fmt.Println("info string illegal move:", parts[j])
				return
			}
		}
	}
}

// goParams holds the parsed arguments of a "go" command.
type goParams struct {
	wtime    int // white's remaining time, ms
	btime    int // black's remaining time, ms
	winc     int // white's increment per move, ms
	binc     int // black's increment per move, ms
	movetime int // explicit per-move time limit, ms (0 = unused)
	depth    int // fixed depth (0 = unused)
	infinite bool // search until "stop"
}

func (s *uciSession) parseGo(parts []string) goParams {
	gp := goParams{}
	for i := 0; i < len(parts); i++ {
		switch parts[i] {
		case "wtime":
			if i+1 < len(parts) {
				gp.wtime, _ = strconv.Atoi(parts[i+1])
				i++
			}
		case "btime":
			if i+1 < len(parts) {
				gp.btime, _ = strconv.Atoi(parts[i+1])
				i++
			}
		case "winc":
			if i+1 < len(parts) {
				gp.winc, _ = strconv.Atoi(parts[i+1])
				i++
			}
		case "binc":
			if i+1 < len(parts) {
				gp.binc, _ = strconv.Atoi(parts[i+1])
				i++
			}
		case "movetime":
			if i+1 < len(parts) {
				gp.movetime, _ = strconv.Atoi(parts[i+1])
				i++
			}
		case "depth":
			if i+1 < len(parts) {
				gp.depth, _ = strconv.Atoi(parts[i+1])
				i++
			}
		case "infinite":
			gp.infinite = true
		}
	}
	return gp
}

// computeTimeLimit decides how long the search should run, in ms. UCI gives
// both players' clocks; the side to move uses its own clock and increment.
// We estimate ~40 moves remaining and use most of the increment while keeping
// a reserve from the base time so it doesn't drain too fast. The slice is
// capped at half the remaining clock to avoid blowing the whole budget on one
// move. A movetime argument overrides everything. Infinite means effectively no
// time limit (the search stops on "stop" or a forced mate).
func (s *uciSession) computeTimeLimit(gp goParams) int64 {
	if gp.movetime > 0 {
		return int64(gp.movetime)
	}
	if gp.infinite || gp.depth > 0 {
		return 1 << 62
	}
	var wtime, winc int
	if s.pos.WhiteToMove {
		wtime, winc = gp.wtime, gp.winc
	} else {
		wtime, winc = gp.btime, gp.binc
	}
	if wtime <= 0 {
		return 1000
	}
	slice := wtime/40 + winc*4/5
	if half := wtime / 2; slice > half {
		slice = half
	}
	if slice < 10 {
		slice = 10
	}
	return int64(slice)
}

// handleGo starts a search in a goroutine and prints "info" + "bestmove"
// when it completes. A "stop" command closes s.stopCh which aborts the
// search; the goroutine then finishes and prints bestmove.
func (s *uciSession) handleGo(parts []string) {
	s.stopSearch()

	gp := s.parseGo(parts)
	timeMs := s.computeTimeLimit(gp)

	// Clone the position for the search goroutine so the main goroutine can
	// keep handling stdin (e.g. "stop") without a data race on the board.
	searchPos := s.pos

	stopCh := make(chan struct{})
	done := make(chan struct{})

	s.searchMu.Lock()
	s.stopCh = stopCh
	s.searchDone = done
	s.searchMu.Unlock()

	go func() {
		defer close(done)
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "search panic: %v\n", r)
				var ml engine.MoveList
				searchPos.LegalMoves(&ml)
				moveStr := "0000"
				if ml.Len() > 0 {
					moveStr = moveToUCI(ml.Get(0))
				}
				os.Stdout.WriteString("bestmove " + moveStr + "\n")
				os.Stdout.Sync()
			}
		}()
		var result ai.SearchResult
		if gp.depth > 0 {
			result = ai.SearchFixedDepthWithTT(&searchPos, gp.depth, stopCh, s.tt)
		} else {
			result = ai.SearchWithTT(&searchPos, int(timeMs), stopCh, s.tt)
		}
		printInfo(result)

		moveStr := moveToUCI(result.Move)
		if result.Move.From == 0 && result.Move.To == 0 {
			var ml engine.MoveList
			searchPos.LegalMoves(&ml)
			if ml.Len() > 0 {
				moveStr = moveToUCI(ml.Get(0))
			}
		}
		os.Stdout.WriteString("bestmove " + moveStr + "\n")
		os.Stdout.Sync()
	}()
}

// printInfo writes a UCI "info" line for the completed search.
func printInfo(r ai.SearchResult) {
	scoreStr := fmt.Sprintf("cp %d", r.Score)
	if r.Score >= 99000 {
		// Mate in N: r.Score is winScore - depth; convert to plies.
		plies := 100000 - r.Score
		mateIn := (plies + 1) / 2
		scoreStr = fmt.Sprintf("mate %d", mateIn)
	} else if r.Score <= -99000 {
		plies := 100000 + r.Score
		mateIn := (plies + 1) / 2
		scoreStr = fmt.Sprintf("mate -%d", mateIn)
	}
	fmt.Printf("info depth %d score %s nodes %d time %d pv %s\n",
		r.Depth, scoreStr, r.Nodes, r.TimeMs, moveToUCI(r.Move))
}

// stopSearch requests any running search to stop and waits for it to
// finish. Safe to call when no search is running (no-op).
func (s *uciSession) stopSearch() {
	s.searchMu.Lock()
	stopCh := s.stopCh
	done := s.searchDone
	s.stopCh = nil
	s.searchDone = nil
	s.searchMu.Unlock()

	if done == nil {
		return
	}
	if stopCh != nil {
		select {
		case <-stopCh:
			// Already closed — no-op.
		default:
			close(stopCh)
		}
	}
	<-done
}

// handleStop closes the active search's stop channel, signaling the
// search goroutine to abort at the next node check.
func (s *uciSession) handleStop() {
	s.searchMu.Lock()
	stopCh := s.stopCh
	s.searchMu.Unlock()
	if stopCh != nil {
		select {
		case <-stopCh:
		default:
			close(stopCh)
		}
	}
}

// ── Main loop ──────────────────────────────────────────────────────

func main() {
	s := newSession()
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		cmd := parts[0]
		args := parts[1:]

		switch cmd {
		case "uci":
			s.handleUci()
		case "isready":
			s.handleIsready()
		case "ucinewgame":
			s.handleUcinewgame()
		case "position":
			s.handlePosition(args)
		case "go":
			s.handleGo(args)
		case "stop":
			s.handleStop()
		case "quit":
			s.stopSearch()
			return
		case "debug":
			// debug on/off — acknowledged, no-op.
		}
		// Unknown commands are ignored per UCI spec.
	}
}