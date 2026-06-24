package main

import (
	"fmt"

	"webassemble/pkg/ai"
)

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