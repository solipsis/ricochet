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

func BenchmarkRenderGif(b *testing.B) {

	b.StopTimer()
	var g *game
	var moves []move
	for {
		g = randomGame()

		g.optimalMoves = g.preCompute(g.activeGoal.position)
		res := g.solve(18)
		moves, _ = parseMoves(res)
		if len(moves) > 10 {
			break
		}
	}
	b.StartTimer()

	// run the Fib function b.N times
	for n := 0; n < b.N; n++ {
		renderGif(g, moves)
	}
}
