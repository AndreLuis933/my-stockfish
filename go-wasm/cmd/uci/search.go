package main

import (
	"fmt"
	"os"

	"webassemble/pkg/ai"
	"webassemble/pkg/engine"
)

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