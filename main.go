package main

import (
	"fmt"
	"strings"
)

type direction uint8

type Goal struct {
	id       byte
	position uint32
}

type game struct {
	size             int
	board            []square
	moves            []move
	robots           map[byte]*robot
	activeRobot      *robot
	goals            []Goal
	activeGoal       Goal
	visits           int
	cache            map[uint32]int
	precomputedMoves []uint32
	id               string

	difficulty         difficulty
	quadrants          []int
	lenOptimalSolution int
}

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
	if len(g.moves) > 0 {
		prevMove := g.moves[len(g.moves)-1]
		isSameRobot := prevMove.id == r.id
		isReverseMovement := prevMove.dir == reverse(dir)

		if isSameRobot && isReverseMovement {
			return false
		}
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

	/* TODO: investigate #4HhM5C1g7B7A67Vy. I think its because the decoding didn't set robot bits
	g.board[r.position] = g.board[r.position] ^ square(ROBOT)
	g.board[end] = g.board[end] ^ square(ROBOT)
	*/
	g.board[r.position] = g.board[r.position] &^ square(ROBOT)
	g.board[end] = g.board[end] | square(ROBOT)

	r.position = end

	return true
}

func (g *game) countRobotBits() {
	count := 0
	for _, b := range g.board {
		if b&square(ROBOT) != 0 {
			count += 1
		}
	}
	fmt.Println(count)
}

func (g *game) search(depth int, maxDepth int) bool {

	// check if game over
	if g.activeRobot.position == g.activeGoal.position {
		return true
	}

	// if too far from optimalMoves needed to get to goal give up
	optimalMoves := int(g.precomputedMoves[g.activeRobot.position])
	if optimalMoves > maxDepth-depth {
		return false
	}

	if depth > maxDepth {
		return false
	}

	// check state cache
	prev, ok := g.cache[g.state()]
	// XXX: Changing this from < to <= fixes incorrect solution for "debugBoard"
	// what is slightly wrong about the original? It was detecting reverse movements
	// of all pieces not just the piece that moved
	if !ok || prev < maxDepth-depth {
		// better than previous
		g.cache[g.state()] = maxDepth - depth
	} else {
		//	fmt.Println("cache hit")
		// we've been here and its worse
		return false
	}

	g.visits += 1

	var breakpoint bool
	/*
		if len(g.moves) >= 2 && g.moves[0].id == 'B' && g.moves[0].dir == UP &&
			g.moves[1].id == 'B' && g.moves[1].dir == RIGHT {
			//g.moves[2].id == 'Y' && g.moves[2].dir == LEFT {
			//	g.moves[3].id == 'Y' && g.moves[3].dir == UP {
			breakpoint = true
		}
	*/

	//for _, id := range []byte{'B', 'Y', 'R', 'G'} {
	//	i := id
	//	r := g.robots[id]

	for i, r := range g.robots {

		for _, dir := range directions {
			prevPosition := r.position

			/*
				if len(g.moves) >= 2 && g.moves[0].id == 'B' && g.moves[0].dir == UP &&
					g.moves[1].id == 'B' && g.moves[1].dir == RIGHT && id == 'Y' && dir == LEFT {
					breakpoint = true
					g.move(r, dir)
				}
			*/

			// attempt to move robot
			if !g.move(r, dir) {

				continue
			}
			g.moves = append(g.moves, move{id: r.id, dir: dir})

			success := g.search(depth+1, maxDepth)

			// undo move
			g.board[prevPosition] = g.board[prevPosition] | square(ROBOT)
			g.board[g.robots[i].position] = g.board[g.robots[i].position] ^ square(ROBOT)
			//g.robots[i] = prevRobot
			r.position = prevPosition

			if success {
				// XXX TODO remove
				if false {
					fmt.Println(breakpoint)
				}
				return true
			}

			// pop from move tracker
			g.moves = g.moves[:len(g.moves)-1]
		}
	}
	return false
}

func (g *game) solve(maxDepth int) string {
	// games are long lived so we want gc to clean up solve cache which won't be used again
	cleanup := func() {
		g.cache = nil
	}
	defer cleanup()

	for currentMaxDepth := 1; currentMaxDepth < maxDepth; currentMaxDepth++ {
		success := g.search(0, currentMaxDepth)
		//fmt.Println("cache-size:", len(g.cache))
		if success {
			var moveStrs []string
			for _, m := range g.moves {
				moveStrs = append(moveStrs, m.String())
			}
			return strings.Join(moveStrs, "-")
		}
	}
	return "no solution in move limit"
}

var directions = []direction{UP, DOWN, LEFT, RIGHT}

func (g *game) state() uint32 {
	/*
		var target uint32 = 0
		var other [3]uint32
		x := 0
		for _, r := range g.robots {
			if r.id == g.activeRobot.id {
				target = r.position
			} else {
				other[x] = r.position
				x++
			}
		}
		if other[0] > other[1] {
			tmp := other[1]
			other[1] = other[0]
			other[0] = tmp
		}
		if other[1] > other[2] {
			tmp := other[2]
			other[2] = other[1]
			other[1] = tmp
		}
		if other[0] > other[1] {
			tmp := other[1]
			other[1] = other[0]
			other[0] = tmp
		}

		s := target
		s |= other[0] << 8
		s |= other[1] << 16
		s |= other[2] << 24
	*/

	s := g.robots['R'].position
	s |= g.robots['B'].position << 8
	s |= g.robots['G'].position << 16
	s |= g.robots['Y'].position << 24

	return s
}

func main() {

	/*
		rg := randomGame()
		out := printBoard(rg.board, rg.size, rg.robots, rg.activeGoal)
		fmt.Println(out)
		fmt.Println(len(out))
		return
	*/

	s := &server{}
	s.run()
	/*
		g := parseBoard(fullBoard)

		start := time.Now()
		var wg sync.WaitGroup
		results := make(chan string)
		rand.Seed(time.Now().UnixNano())
		for idx, ig := range g.goals {
			wg.Add(1)
			go func(target Goal, index int) {
				defer wg.Done()
				cpy := g.clone()
				cpy.activeGoal = target
				//		cpy.activeGoal.position = uint32(rand.Intn(255))
				cpy.activeRobot = cpy.robots[target.id]
				cpy.optimalMoves = cpy.preCompute(cpy.activeGoal.position)
				res := cpy.solve(20)
				results <- fmt.Sprintf("Puzzle: %d -> %s", index, res)

			}(ig, idx)
		}

		done := make(chan struct{})
		go func() {
			for msg := range results {
				fmt.Println(msg)
			}
			close(done)
		}()

		wg.Wait()
		stop := time.Since(start)
		close(results)
		<-done
		fmt.Printf("Total Time: %s \n", stop)

		printBoard(g.board, g.size, g.robots, g.activeGoal)
	*/

}

func (g *game) setRobot(id byte, pos uint32) {
	for idx, v := range g.robots {
		if v.id == id {
			g.robots[idx].position = pos
		}
	}
}

func parseBoard(input string, quadrants []int) game {
	/*(
		input := `???---???---???---???
	| R     | B |
	???   ???   ???   ???
	|     r     |
	???   ???   ???   ???
	|         G |
	???---???---???---???`

		input = fullBoard
	*/

	// I can do smarter parsing without the edge cases by always working in 3 part rows
	// just only advance the row pointer by 2
	// read everything into buffer initially

	// TODO: trim trailing newlines when input comes from files
	input = strings.TrimSpace(input)
	lines := strings.Split(input, "\n")
	size := len(strings.Split(lines[0], "---")) - 1

	board := make([]square, size*size)
	robots := make(map[byte]*robot)
	goals := make([]Goal, 0)

	for row := 0; row < size; row++ {
		for col := 0; col < size; col++ {
			// top
			tLine := lines[row*2]
			if tLine[(col*6)+5] == '-' { // weird math cause 3byte utf8 char
				board[(row*size)+col] = board[(row*size)+col] | square(UP)
			}
			// bottom
			bLine := lines[(row*2)+2]
			if bLine[(col*6)+5] == '-' {
				board[(row*size)+col] = board[(row*size)+col] | square(DOWN)
			}
			// left
			mLine := lines[(row*2)+1]
			if mLine[col*4] == '|' {
				board[(row*size)+col] = board[(row*size)+col] | square(LEFT)
			}
			// right
			if mLine[(col+1)*4] == '|' {
				board[(row*size)+col] = board[(row*size)+col] | square(RIGHT)
			}
			//center
			if mLine[(col*4)+2] != ' ' {
				c := mLine[(col*4)+2]
				if c >= 'A' && c <= 'Z' {
					board[(row*size)+col] = board[(row*size)+col] | square(ROBOT)
					robots[c] = &robot{id: c, position: uint32((row * size) + col)}
				}
				if c >= 'a' && c <= 'z' {
					goals = append(goals, Goal{id: c - 32, position: uint32((row * size) + col)})
				}
			}
		}
	}

	g := game{
		size:        size,
		board:       board,
		robots:      robots,
		goals:       goals,
		activeRobot: robots['R'],
		activeGoal:  goals[0],
		cache:       make(map[uint32]int),
		quadrants:   quadrants,
	}

	return g
}

func (g *game) clone() game {
	board := make([]square, len(g.board))
	copy(board, g.board)

	robots := make(map[byte]*robot)
	for _, r := range g.robots {
		robots[r.id] = &robot{id: r.id, position: r.position}
	}

	goals := make([]Goal, len(g.goals))
	copy(goals, g.goals)

	ng := game{
		size:        g.size,
		board:       board,
		robots:      robots,
		moves:       make([]move, 0),
		cache:       make(map[uint32]int),
		visits:      0,
		activeRobot: robots[g.activeRobot.id],
		goals:       goals,
		activeGoal:  g.activeGoal,
		id:          g.id,
	}
	return ng
}
