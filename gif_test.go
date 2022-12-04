package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestRenderGif(t *testing.T) {
	g := randomGame()

	g.optimalMoves = g.preCompute(g.activeGoal.position)
	res := g.solve(18)
	moves, _ := parseMoves(res)

	var moveStrs []string
	for _, m := range g.moves {
		moveStrs = append(moveStrs, m.String())
	}
	fmt.Println("Optimal:", strings.Join(moveStrs, "-"))

	renderGif(g, moves)

}
