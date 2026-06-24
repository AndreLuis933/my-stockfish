package main

import (
	"bufio"
	"os"
	"strings"
	"sync"

	"webassemble/pkg/engine"
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