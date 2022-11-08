package main

import (
	"fmt"
	"strings"
)

func printBoard(board []square, size int, robots map[byte]*robot, goal Goal) {
	var b strings.Builder

	// one row at a time
	for row := 0; row < size; row += 1 {
		// top
		b.WriteRune('•')
		for col := 0; col < size; col += 1 {
			if board[(row*size)+col]&square(UP) != 0 {
				b.WriteString("---")
			} else {
				b.WriteString("   ")
			}
			b.WriteRune('•')
		}
		b.WriteString("\n")

		b.WriteString("|")
		// mid
		for col := 0; col < size; col += 1 {
			// TODO: need to check for robots/goals
			b.WriteString("   ")
			if board[(row*size)+col]&square(RIGHT) != 0 {
				b.WriteString("|")
			} else {
				b.WriteString(" ")
			}
		}
		b.WriteString("\n")
	}
	// bottom row guaranteed to be all lines
	b.WriteRune('•')
	for col := 0; col < size; col += 1 {
		b.WriteString("---")
		b.WriteRune('•')
	}

	fmt.Println(b.String())
}
