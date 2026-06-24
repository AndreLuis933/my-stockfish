package main

import "strconv"

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