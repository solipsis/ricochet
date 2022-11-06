package main

// calculate the minimal number of moves it would take to get to the target
// if the moving piece could move like a rook (stopping arbitrarily). This helps greatly prune
// the search space
func (g *game) preCompute(target uint32) []uint32 {

	active := make([]bool, len(g.board))
	optimalMoves := make([]uint32, len(g.board))

	for idx := range optimalMoves {
		optimalMoves[idx] = 999999
	}

	optimalMoves[target] = 0
	active[target] = true
	done := false

	for !done {
		done = true

		for idx, _ := range g.board {
			if !active[idx] {
				continue
			}
			active[idx] = false

			score := optimalMoves[idx] + 1
			for _, dir := range directions {
				curSquare := idx

				// go until we hit a wall
				for {
					if g.hasWall(uint32(curSquare), dir) {
						break
					}

					curSquare += g.offset(dir)
					if optimalMoves[curSquare] > score {
						optimalMoves[curSquare] = score
						active[curSquare] = true
						done = false
					}
				}
			}
		}
	}

	/*
		for x := 0; x < g.size; x++ {
			fmt.Printf("\n")
			for y := 0; y < g.size; y++ {
				if optimalMoves[(x*g.size)+y] > 20 {
					fmt.Printf("X")
				} else {
					fmt.Printf("%d", optimalMoves[(x*g.size)+y])
				}
			}
		}
	*/
	return optimalMoves
}
