package main

import (
	"fmt"

	"github.com/njones/base58"
)

// Q1 Q2 Q3 Q4 R1 R2 R3 R4 GL GC ROT Extra
func encode(g *game) (string, error) {
	if g == nil {
		return "", fmt.Errorf("nil game")
	}

	if g.quadrants == nil || len(g.quadrants) != 4 {
		return "", fmt.Errorf("Unexpected board quadrants for encoding")
	}

	if len(g.robots) != 4 {
		return "", fmt.Errorf("Unexpected num robots for encoding")
	}

	encoded := make([]byte, 12)
	// map
	encoded[0] = byte(g.quadrants[0])
	encoded[1] = byte(g.quadrants[1])
	encoded[2] = byte(g.quadrants[2])
	encoded[3] = byte(g.quadrants[3])
	// robots
	encoded[4] = byte(g.robots['R'].position)
	encoded[5] = byte(g.robots['G'].position)
	encoded[6] = byte(g.robots['B'].position)
	encoded[7] = byte(g.robots['Y'].position)
	// goal
	encoded[8] = byte(g.activeGoal.position)
	encoded[9] = byte(g.activeGoal.id)
	// rotation
	encoded[10] = 0
	// extra
	encoded[11] = 0

	encoding := base58.StdEncoding.EncodeToString(encoded)

	return encoding, nil
}

func decode(id string) (*game, error) {
	buf, err := base58.StdEncoding.DecodeString(id)
	if err != nil {
		return nil, err
	}

	q1, q2, q3, q4 := buf[0], buf[1], buf[2], buf[3]
	boardStr, err := boardFromQuadrants(int(q1), int(q2), int(q3), int(q4))
	if err != nil {
		return nil, fmt.Errorf("boardFromQuadrants: %v", err)
	}
	g := parseBoard(boardStr, []int{int(q1), int(q2), int(q3), int(q4)})
	//g.quadrants = []int{int(q1), int(q2), int(q3), int(q4)}

	// reset any robot positions from board string tile
	for idx := range g.board {
		g.board[idx] = g.board[idx] &^ square(ROBOT)
	}
	g.robots['R'].position = uint32(buf[4])
	g.robots['G'].position = uint32(buf[5])
	g.robots['B'].position = uint32(buf[6])
	g.robots['Y'].position = uint32(buf[7])
	g.activeGoal.position = uint32(buf[8])
	g.activeGoal.id = buf[9]
	g.id = id

	for _, r := range g.robots {
		g.board[r.position] = g.board[r.position] | square(ROBOT)
	}

	//fmt.Println(printBoard(g.board, g.size, g.robots, g.activeGoal))

	return &g, nil
}
