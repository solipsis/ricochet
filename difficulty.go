package main

type difficulty int

const (
	UNKNOWN difficulty = iota
	EASY
	MEDIUM
	HARD
	EXTREME
)

func (d difficulty) String() string {
	switch d {
	case EASY:
		return "easy"
	case MEDIUM:
		return "medium"
	case HARD:
		return "hard"
	case EXTREME:
		return "extreme"
	default:
		return "unknown"
	}
}
