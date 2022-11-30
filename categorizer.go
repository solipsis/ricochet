package main

import (
	"math/rand"
	"strings"
	"time"
)

type categorizer struct {
	easy    chan (*game)
	medium  chan (*game)
	hard    chan (*game)
	extreme chan (*game)
}

func weightSolution(solution string) int {
	moves := strings.Split(solution, "-")
	//TODO: count how many unique robots used? or is just moves better
	return len(moves)
}

// easy 5 < 6
// medium 7 12
// hard 12+

var possibleRobots = []byte{'R', 'G', 'B', 'Y'}

// load a board
// select random goal (no reason I can't randomize target robot?)
// select random starting locations

func randomGame() *game {
	//g := parseBoard(fullBoard)
	g := parseBoard(randomBoard())
	// select random goal
	rand.Seed(time.Now().UnixNano())
	goalIdx := rand.Intn(len(g.goals))

	// pick a goal location and a random color for that goal
	g.activeGoal = g.goals[goalIdx]
	g.activeGoal.id = possibleRobots[rand.Intn(4)]
	g.activeRobot = g.robots[g.activeGoal.id]

	// randomly place robots
	// can't be where another robot is
	// can't be on goal
	// TODO: can't be on middle / maybe X character marks invalid squares

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

	return &g
}
