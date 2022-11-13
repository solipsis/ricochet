package main

import (
	"fmt"
	"strings"
)

func validate(g *game, board []square, moves []move, goal Goal) bool {
	cpy := g.clone()

	// do all moves
	for _, m := range moves {
		cpy.move(cpy.robots[m.id], m.dir)
	}

	// check that target is on the goal
	return cpy.robots[goal.id].position == goal.position
}

func parseMoves(in string) ([]move, error) {
	var moves []move

	in = strings.TrimSpace(in)
	f := func(r rune) bool {
		return r == ' ' || r == '-'
	}
	parts := strings.FieldsFunc(in, f)

	// reject pathalogical inputs
	if len(parts) > 30 {
		return moves, fmt.Errorf("too many moves... rejecting")
	}

	for _, p := range parts {
		m, err := parseMove(p)
		if err != nil {
			return nil, err
		}
		moves = append(moves, m)
	}

	return moves, nil
}

func parseMove(in string) (move, error) {
	if len(in) != 2 {
		return move{}, fmt.Errorf("invalid move")
	}
	m := move{}
	upper := strings.ToUpper(in)
	switch upper[0] {
	case 'R', 'Y', 'G', 'B':
		m.id = upper[0]
	default:
		return move{}, fmt.Errorf("invalid robot ID")
	}
	switch upper[1] {
	case 'U':
		m.dir = UP
	case 'D':
		m.dir = DOWN
	case 'L':
		m.dir = LEFT
	case 'R':
		m.dir = RIGHT
	default:
		return move{}, fmt.Errorf("invalid move direction")
	}
	return m, nil
}
