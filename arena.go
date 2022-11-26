package main

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

func tokenReward(difficulty string) int {
	switch difficulty {
	case "easy":
		return 10
	case "medium":
		return 15
	case "hard":
		return 20
	default:
		return 0
	}
}

func arenaSolution(dg *discordgo.Session, i *discordgo.Interaction, instance *discordInstance, moves []move) error {

	// if first + optimal tokens
	// if first tokens
	// if first optimal tokens

	firstSolve := instance.solutionTracker.numSubmitted() == 0
	isOptimal := len(moves) == instance.activeGame.lenOptimalSolution
	tokensEarned := 0
	if firstSolve {
		tokensEarned += tokenReward(instance.activeGame.difficultyName)
	}
	if isOptimal {
		// bonus points if first optimal solution
		if instance.solutionTracker.numSubmitted() == 0 || len(moves) < len(instance.solutionTracker.currentBest()) {
			tokensEarned += tokenReward(instance.activeGame.difficultyName)
		}
	}

	instance.solutionTracker.set(i.Member.User.ID, moves)

	var content string
	if isOptimal {
		content = fmt.Sprintf("<@%s> solved with an :tada:**optimal**:tada: %d move solution", i.Member.User.ID, len(moves))
	} else {
		content = fmt.Sprintf("<@%s> solved with a %d move solution", i.Member.User.ID, len(moves))
	}
	if tokensEarned > 0 {
		content = fmt.Sprintf("%s +%d :arena:", content, tokensEarned)
	}
	if _, err := dg.ChannelMessageSend(i.ChannelID, content); err != nil {
		log.Printf("Sending arena solution message: %v", err)
		return err
	}

	return nil
}
