package main

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
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

func TestExtreme(t *testing.T) {

	var wg sync.WaitGroup
	for x := 0; x < 10; x++ {
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				g := parseBoard(extremeBoard)
				rand.Seed(time.Now().UnixNano())
				goalIdx := rand.Intn(len(g.goals))
				g.activeGoal = g.goals[goalIdx]
				g.activeGoal.id = possibleRobots[rand.Intn(4)]
				g.activeRobot = g.robots[g.activeGoal.id]

				// grab random squares for each robot that aren't a goal tile
				possibleSquares := make([]uint32, g.size*g.size)
				for i := 0; i < g.size*g.size; i++ {
					possibleSquares[i] = uint32(i)
				}
				rand.Shuffle(len(possibleSquares), func(i, j int) { possibleSquares[i], possibleSquares[j] = possibleSquares[j], possibleSquares[i] })

				for _, robot := range g.robots {
					// toggle off existing robot bit
					g.board[robot.position] = g.board[robot.position] &^ square(ROBOT)

					// grab a random square from candidate list,
					// try again if grabbed square is invalid
					for {
						pop := possibleSquares[len(possibleSquares)-1]
						possibleSquares = possibleSquares[:len(possibleSquares)-1]

						// TODO: check other reasons a spot may be invalid i.e middle
						if pop != g.activeGoal.position {
							g.board[pop] |= square(ROBOT)
							robot.position = pop
							break
						}
					}
				}
				g.optimalMoves = g.preCompute(g.activeGoal.position)
				res := g.solve(20)
				fmt.Println(res)
			}()
		}
		wg.Wait()
	}

}

func TestRandomGame(t *testing.T) {
	g := randomGame()
	printBoard(g.board, g.size, g.robots, g.activeGoal)

	g.optimalMoves = g.preCompute(g.activeGoal.position)
	res := g.solve(15)
	fmt.Println(res)
}
