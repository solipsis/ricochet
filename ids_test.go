package main

import (
	"fmt"
	"testing"
)

func TestEncodeDecode(t *testing.T) {

	g := randomGame()
	str, err := encode(g)
	if err != nil {
		t.Fatal(err)
	}
	printed := printBoard(g.board, g.size, g.robots, g.activeGoal)

	// decode and make sure we get same state
	g2, err := decode(str)
	if err != nil {
		t.Fatal(err)
	}
	printed2 := printBoard(g2.board, g2.size, g2.robots, g2.activeGoal)

	str2, err := encode(g2)
	if err != nil {
		t.Fatal(err)
	}

	if str != str2 {
		t.Fatalf("encoding after decoding is different")
	}

	if printed != printed2 {
		t.Fatalf("printed boards don't match")
	}

}

func TestWeirdDecode(t *testing.T) {

	start := "3BxvKmWMqjKASyDq"

	g, err := decode(start)
	if err != nil {
		t.Fatal(err)
	}

	id, err := encode(g)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("start: %s, g.id: %s - decoded: %s\n", start, g.id, id)

}
