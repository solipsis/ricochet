package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
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

var addRewardLedgerEntryQuery = `
	INSERT INTO public."TokenLedger"
	(id, "createdAt", "updatedAt", "amount", "userId", "type", "data", "uniqueId")
	VALUES($1, $2, $3, $4, $5, $6, $7, $8)
`

func addSolveRewardLedgerEntry(conn *pgxpool.Pool, userID string, amount int) error {
	timestamp := time.Now()
	uniqueID := uuid.New()

	tx, err := conn.Begin(context.TODO())
	if err != nil {
		return fmt.Errorf("starting transaction: %v", err)
	}

	_, err = tx.Exec(context.Background(), addRewardLedgerEntryQuery,
		fmt.Sprintf("RR-%s", uniqueID),
		timestamp,
		timestamp,
		amount,
		userID,
		"MANUAL",
		`{"reason": "ricochet-robotbot"}`,
		uniqueID,
	)
	if err != nil {
		return fmt.Errorf("executing discord reward query: %v\n", err)
	}
	/*
		fmt.Println(userID, amount, tag.RowsAffected(), tag.String())
		spew.Dump(tag)
		spew.Dump(conn)
	*/

	// commit the result
	err = tx.Commit(context.TODO())
	if err != nil {
		return fmt.Errorf("committing transaction: %v", err)
	}

	return nil
}

func arenaSolution(dg *discordgo.Session, i *discordgo.Interaction, instance *discordInstance, db *pgxpool.Pool, moves []move) error {

	currentSolutions := instance.getSolutions(instance.puzzleIdx)

	firstSolve := currentSolutions.numSubmitted() == 0
	isOptimal := len(moves) == instance.activeGame.lenOptimalSolution
	tokensEarned := 0
	if firstSolve {
		tokensEarned += tokenReward(instance.activeGame.difficultyName)
	}
	if isOptimal {
		// bonus points if first optimal solution
		if currentSolutions.numSubmitted() == 0 || len(moves) < len(currentSolutions.currentBest()) {
			tokensEarned += tokenReward(instance.activeGame.difficultyName)
		}
	}

	current := currentSolutions.get(i.Member.User.ID)
	if len(current) == 0 || len(moves) <= len(current) {
		currentSolutions.set(i.Member.User.ID, moves)
	}

	// only print solve messages if there is not an active tournament
	if instance.activeTournament == nil {
		var content string
		if isOptimal {
			content = fmt.Sprintf("<@%s> solved with an :tada:**optimal**:tada: %d move solution", i.Member.User.ID, len(moves))
		} else {
			content = fmt.Sprintf("<@%s> solved with a %d move solution", i.Member.User.ID, len(moves))
		}
		if tokensEarned > 0 {
			content = fmt.Sprintf("%s +%d <:arena:917512583160930364>", content, tokensEarned)
		}
		if _, err := dg.ChannelMessageSend(i.ChannelID, content); err != nil {
			log.Printf("Sending arena solution message: %v", err)
			return err
		}
	}

	// look up linked arena account if exists
	arenaID, err := arenaIDFromLinkedAccount(db, i.Member.User.ID)
	if err != nil || arenaID == "" {
		log.Printf("Solver does not have a linked arena account")
		return nil
	}

	// add token reward to arena db
	if err := addSolveRewardLedgerEntry(db, arenaID, tokensEarned); err != nil {
		return fmt.Errorf("Unable to add reward to ledger: %v", err)
	}

	return nil
}

var linkedAccountQuery = `
	SELECT a."userId"
	FROM "Account" a
	WHERE a."providerAccountId" = $1 AND a."provider" = 'discord';
`

func arenaIDFromLinkedAccount(conn *pgxpool.Pool, discordUserID string) (string, error) {
	var userID sql.NullString
	err := conn.QueryRow(context.Background(), linkedAccountQuery, discordUserID).Scan(&userID)
	if err != nil {
		return "", err
	}
	return userID.String, err
}
