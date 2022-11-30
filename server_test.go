package main

import "testing"

func TestServer(t *testing.T) {

	s := &server{}
	s.run()
}

func TestLookForSolutions(t *testing.T) {
	s := &server{}
	cat := categorizer{
		easy:    make(chan (*game), gameBuffer),
		medium:  make(chan (*game), gameBuffer),
		hard:    make(chan (*game), gameBuffer),
		extreme: make(chan (*game), gameBuffer),
	}
	s.categorizer = &cat

	lookForSolutions(s)
}
