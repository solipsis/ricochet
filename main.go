package main

import "fmt"

type direction uint8

const (
	UP direction = 1 << iota
	DOWN
	LEFT
	RIGHT
	ROBOT
)

type robot struct {
	position uint32
	id       byte
}

type move struct {
	id  byte
	dir direction
}

func (m *move) String() string {
	//fmt.Printf("id: %c, dir: %s\n", rune(m.id), m.dir)
	return fmt.Sprintf("%c%s", m.id, m.dir)
}

func (d direction) String() string {
	switch d {
	case UP:
		return "U"
	case DOWN:
		return "D"
	case LEFT:
		return "L"
	case RIGHT:
		return "R"
	default:
		panic("invalid direction")
	}
}

func reverse(d direction) direction {
	switch d {
	case UP:
		return DOWN
	case DOWN:
		return UP
	case LEFT:
		return RIGHT
	case RIGHT:
		return LEFT
	default:
		panic("invalid direction")
	}
}

type square uint32

func (g *game) offset(d direction) int {
	switch d {
	case UP:
		return g.size * -1
	case DOWN:
		return g.size
	case LEFT:
		return -1
	case RIGHT:
		return 1
	default:
		panic("invalid direction")
	}

}

func (g *game) hasWall(loc uint32, dir direction) bool {
	switch dir {
	case UP:
		return g.board[loc]&square(UP) != 0
	case DOWN:
		return g.board[loc]&square(DOWN) != 0
	case RIGHT:
		return g.board[loc]&square(RIGHT) != 0
	case LEFT:
		return g.board[loc]&square(LEFT) != 0
	default:
		panic("invalid direction")
	}
}

func (g *game) move(r *robot, dir direction) bool {
	if g.hasWall(r.position, dir) {
		return false
	}
	// if move is reverse of the last move we did, abort
	if len(g.moves) > 0 && g.moves[len(g.moves)-1].dir == reverse(dir) {
		return false
	}

	// if next square has robot, abort
	next := uint32(int(r.position) + g.offset(dir))
	if g.board[next]&square(ROBOT) != 0 {
		return false
	}

	end := next
	// go until we hit a wall in the current square or there is a robot in next square
	for {
		if g.hasWall(end, dir) {
			break
		}
		// if next square has robot, abort
		next := uint32(int(end) + g.offset(dir))
		if g.board[next]&square(ROBOT) != 0 {
			break
		}
		end = next
	}

	g.board[r.position] = g.board[r.position] ^ square(ROBOT)
	g.board[end] = g.board[end] ^ square(ROBOT)
	r.position = end

	return true
}

func (g *game) search(depth int, maxDepth int) bool {
	// check if game over
	if g.robots[0].position == goal {
		return true
	}

	// check precompute

	// check state cache

	if depth > maxDepth {
		return false
	}

	g.visits += 1

	for i, r := range g.robots {
		for _, dir := range directions {
			prevRobot := r

			// attempt to move robot
			if !g.move(&g.robots[i], dir) {
				continue
			}
			g.moves = append(g.moves, move{id: r.id, dir: dir})

			success := g.search(depth+1, maxDepth)

			// undo move
			g.board[prevRobot.position] = g.board[prevRobot.position] | square(ROBOT)
			g.board[g.robots[i].position] = g.board[g.robots[i].position] ^ square(ROBOT)
			g.robots[i] = prevRobot

			if success {
				return true
			}

			// pop from move tracker
			g.moves = g.moves[:len(g.moves)-1]
		}
	}
	return false
}

func (g *game) solve(maxDepth int) {
	for currentMaxDepth := 1; currentMaxDepth < maxDepth; currentMaxDepth++ {
		success := g.search(0, currentMaxDepth)
		if success {
			fmt.Println("yay")
			for _, m := range g.moves {
				fmt.Println(m.String())
			}
			fmt.Println(g.visits)
			break
		}
	}
}

var directions = []direction{UP, DOWN, LEFT, RIGHT}

type game struct {
	size   int
	board  []square
	moves  []move
	robots []robot
	visits int
}

const goal = 2

func main() {
	g := game{
		size: 3,
		board: []square{
			0 | square(UP) | square(LEFT),
			0 | square(UP) | square(RIGHT),
			0 | square(UP) | square(LEFT) | square(RIGHT) | square(ROBOT),
			0 | square(LEFT),
			0,
			0 | square(RIGHT),
			0 | square(LEFT) | square(DOWN),
			0 | square(DOWN),
			0 | square(RIGHT) | square(DOWN) | square(ROBOT),
		},
		robots: []robot{
			{position: 0, id: 'R'},
			{position: 2, id: 'B'},
			{position: 8, id: 'G'},
		},
	}
	g.solve(5)
}
