package main

import (
	"fmt"
	"strings"

	"webassemble/pkg/engine"
)

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