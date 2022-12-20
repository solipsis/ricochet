package main

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
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
	boardStr, quadrants := randomBoard()
	g := parseBoard(boardStr, quadrants)
	//g.quadrants = quandrants
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
	id, err := encode(&g)
	if err != nil {
		log.Printf("unable to encode random game: %v", err)
	}
	g.id = id

	return &g
}

func lookForSolutions(s *server) {
	s.isSearching = true // not thread-safe but probably will never matter
	var wg sync.WaitGroup
	for x := 0; x < 4; x++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				if len(s.categorizer.easy) == gameBuffer && len(s.categorizer.medium) == gameBuffer && len(s.categorizer.hard) == gameBuffer {
					break
				}

				rg := randomGame()
				rg.precomputedMoves = rg.preCompute(rg.activeGoal.position)
				res := rg.solve(20)
				moves, _ := parseMoves(res)
				numMoves := len(moves)
				rg.lenOptimalSolution = numMoves

				// Add solution to proper buffer. Discard if that buffer already has enough solutions
				// of that length
				if numMoves >= 6 && numMoves <= 8 {
					select {
					case s.categorizer.easy <- rg:
						rg.difficulty = EASY
						fmt.Println("Easy found:")
					default:
						//			fmt.Println("discarding easy")
					}
				} else if numMoves >= 9 && numMoves <= 12 {
					select {
					case s.categorizer.medium <- rg:
						rg.difficulty = MEDIUM
						fmt.Println("Medium found:")
					default:
						//			fmt.Println("discarding medium")
					}
				} else if numMoves >= 13 && numMoves <= 16 {
					select {
					case s.categorizer.hard <- rg:
						rg.difficulty = HARD
						fmt.Println("Hard found:", numMoves)
					default:
						fmt.Println("Hard found:", numMoves)
						//			fmt.Println("discarding hard")
					}
				} else if numMoves >= 17 && numMoves <= 20 {
					fmt.Println("EXTREME found:", numMoves)
					select {
					case s.categorizer.extreme <- rg:
						rg.difficulty = EXTREME
						fmt.Println("EXTREME found:", numMoves)
					default:
						//			fmt.Println("discarding hard")
					}
				}

			}
		}()
	}
	wg.Wait()
	s.isSearching = false

}
