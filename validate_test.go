package main

import (
	"fmt"
	"testing"
)

func TestParse(t *testing.T) {

	input := "RU-RD-BL-br-gU-gd-gl-gr-yu-yd-yl-yr"
	out, err := parseMoves(input)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(out)
}

func TestValidate(t *testing.T) {

	input := `
•---•---•---•
| R     | B |
•   •   •   •
|     r     |
•   •   •   •
|         G |
•---•---•---•`

	g := parseBoard(input)
	printBoard(g.board, g.size, g.robots, g.activeGoal)

	moves, err := parseMoves("GL-BD-BL-RR-RR-RD")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(validate(&g, g.board, moves, g.activeGoal))
}

func TestBroken(t *testing.T) {
	g := parseBoard(debugBoard)
	g.optimalMoves = g.preCompute(g.activeGoal.position)
	g.activeRobot = g.robots['B']
	fmt.Println(printBoard(g.board, g.size, g.robots, g.activeGoal))
	res := g.solve(9)

	fmt.Println(res)

}

func TestRandomGame(t *testing.T) {
	g := randomGame()
	printBoard(g.board, g.size, g.robots, g.activeGoal)

	g.optimalMoves = g.preCompute(g.activeGoal.position)
	res := g.solve(15)
	fmt.Println(res)
}
