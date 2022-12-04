package main

import (
	"bytes"
	"fmt"
	"image/png"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

var tournamentTemplate = `%s used **/tournament**

**3** puzzles will be shown in a row and you will have **%d** minutes to solve each one.
The winner is the user with the fewest total moves across all puzzles.
Any puzzle you do not solve in the time limit will count as **30** moves.

Tournament begins: **<t:%d:R>**`

//var tournamentPuzzleTime = time.Second * 15
var tournamentStartTime = time.Second * 60

var defaultTournamentDuration = 3

type tournament struct {
	games []tournamentGame
}

type tournamentGame struct {
	g     *game
	id    int
	index int
}

func (s *server) handleTournament(dg *discordgo.Session, i *discordgo.InteractionCreate) error {
	err := dg.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: 1 << 6, // ephemeral
		},
	})
	if err != nil {
		log.Printf("repsonding with deferred ack: %v", err)
		return fmt.Errorf("repsonding with deferred ack: %v", err)
	}

	// Parse command options
	// TODO: difficulty enum
	difficulty := "medium" // default
	var durationMinutes int
	for _, opt := range i.Interaction.ApplicationCommandData().Options {
		if opt.Name == "difficulty" {
			switch opt.Value.(string) {
			case "easy":
				difficulty = "easy"
			case "medium":
				difficulty = "medium"
			case "hard":
				difficulty = "hard"
			default:
				difficulty = "medium"
			}
		}
		if opt.Name == "duration" {
			durationMinutesStr := opt.Value.(string)
			parsed, err := strconv.ParseInt(durationMinutesStr, 10, 64)
			if err != nil {
				durationMinutes = defaultTournamentDuration
			} else {
				durationMinutes = int(parsed)
			}

			if durationMinutes < 1 {
				durationMinutes = 1
			}
			if durationMinutes > 10 {
				durationMinutes = 10
			}
		}
	}
	if durationMinutes == 0 {
		durationMinutes = defaultTournamentDuration
	}

	//	fmt.Printf("difficulty: %s, duration: %d\n", difficulty, durationMinutes)

	// look up instance
	instance := s.instances[i.GuildID]

	// haven't solved current puzzle
	if instance.activeGame != nil {
		optimalFound := len(instance.getSolutions(instance.puzzleIdx).currentBest()) == instance.activeGame.lenOptimalSolution
		timePassed := time.Since(instance.puzzleTimestamp) > (time.Second * 60 * 5)
		if !optimalFound && !timePassed {
			content := "Current puzzle must be solved optimally or 5 minutes have passed before requesting a new one"
			dg.InteractionResponseEdit(i.Interaction,
				&discordgo.WebhookEdit{
					Content: &content,
				},
			)
			return nil
		}
	}
	if instance.activeTournament != nil {
		content := ":x: There is already an active tournament"
		dg.InteractionResponseEdit(i.Interaction,
			&discordgo.WebhookEdit{
				Content: &content,
			},
		)
		return nil
	}

	instance.activeTournament = &tournament{}

	// serve welcome message
	var displayName string
	if i.Interaction.Member.Nick != "" {
		displayName = i.Interaction.Member.Nick
	} else {
		displayName = i.Interaction.Member.User.Username
	}
	tournyText := fmt.Sprintf(tournamentTemplate, displayName, durationMinutes, time.Now().Add(tournamentStartTime).Unix())
	_, err = dg.ChannelMessageSend(instance.channelID, tournyText)
	if err != nil {
		content := ":x: Unable to create tournament, please try again later"
		dg.InteractionResponseEdit(i.Interaction,
			&discordgo.WebhookEdit{
				Content: &content,
			},
		)
		return fmt.Errorf("creating tournament: %v", err)
	}
	time.Sleep(tournamentStartTime)

	// serve 3 puzzles one at a time
	numPuzzles := 3
	for x := 0; x < numPuzzles; x++ {
		var g *game
		/*
			if x == numPuzzles-1 { // harder puzzle for final
				// TODO: change both back
				g = <-s.categorizer.easy
			} else {
				g = <-s.categorizer.easy
			}
		*/
		switch difficulty {
		case "easy":
			g = <-s.categorizer.easy
		case "medium":
			g = <-s.categorizer.medium
		case "hard":
			g = <-s.categorizer.hard
		default:
			g = <-s.categorizer.medium
		}

		if !s.isSearching {
			go lookForSolutions(s)
		}

		instance.activeGame = g
		instance.puzzleTimestamp = time.Now()

		var moveStrs []string
		for _, m := range g.moves {
			moveStrs = append(moveStrs, m.String())
		}
		fmt.Println("Optimal:", strings.Join(moveStrs, "-"))

		img, err := render(g)
		if err != nil {
			cancelTournament(dg, instance, instance.activeTournament)
			return fmt.Errorf("rendering board: %v", err)
		}

		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return fmt.Errorf("encoding board image: %v", err)
		}
		instance.puzzleIdx += 1

		file := &discordgo.File{
			Name:        "board.png",
			ContentType: "image/png",
			Reader:      &buf,
		}

		tg := tournamentGame{g: instance.activeGame, id: instance.puzzleIdx, index: x}
		instance.activeTournament.games = append(instance.activeTournament.games, tg)
		_, err = dg.ChannelMessageSendComplex(instance.channelID, &discordgo.MessageSend{
			Content: tournamentPuzzleContent(i.Interaction.Member, tg, time.Now().Add(time.Second*60*time.Duration(durationMinutes))),
			Files:   []*discordgo.File{file},
		})
		if err != nil {
			content := ":x: Unable to create puzzle, please try again later"
			dg.InteractionResponseEdit(i.Interaction,
				&discordgo.WebhookEdit{
					Content: &content,
				},
			)
			return fmt.Errorf("uploading puzzle to discord: %v", err)
		}
		time.Sleep(time.Second * 60 * time.Duration(durationMinutes))
	}

	endTournament(dg, instance, instance.activeTournament)

	instance.activeTournament = nil

	return nil
}

func endTournament(dg *discordgo.Session, instance *discordInstance, t *tournament) error {
	instance.activeTournament = nil
	instance.activeGame = nil

	// find unique users across all puzzles in tournament
	userIDs := make(map[string]bool)
	for _, tg := range t.games {
		users := instance.getSolutions(tg.id).users()
		for _, u := range users {
			userIDs[u] = true
		}
	}

	// no one solved anything. Just end the tournament
	if len(userIDs) == 0 {
		content := "everyone lost the tournament :cry:"
		_, err := dg.ChannelMessageSend(instance.channelID, content)
		if err != nil {
			return fmt.Errorf("ending empty tournament: %v", err)
		}
		return nil
	}

	//spew.Dump(instance.solutions)
	scores := make(map[string][]int)
	for _, tg := range t.games {
		solutions := instance.getSolutions(tg.id)
		for userID, _ := range userIDs {
			// user didn't solve this one so apply penalty
			moves := solutions.get(userID)
			if moves == nil {
				scores[userID] = append(scores[userID], 30)
			} else {
				scores[userID] = append(scores[userID], len(moves))
			}
		}
	}

	/*
		// testing bot users
		for z := 0; z < 10; z++ {
			scores[fmt.Sprintf("%d", z)] = []int{rand.Intn(30), rand.Intn(30), rand.Intn(30)}
		}
	*/

	// sort scores
	type tournamentScore struct {
		userID string
		total  int
		scores []int
	}
	var tournamentScores []tournamentScore
	for userID, userScores := range scores {

		var total int
		for _, v := range userScores {
			total += v
		}

		tournamentScores = append(tournamentScores, tournamentScore{
			userID: userID,
			total:  total,
			scores: userScores,
		})
	}
	sort.Slice(tournamentScores, func(i, j int) bool {
		return tournamentScores[i].total < tournamentScores[j].total
	})

	// build leaderboard
	var sb strings.Builder
	sb.WriteString("**Tournament Results:**\n")
	position := 0
	bestScore := -1
	for _, ts := range tournamentScores {
		if ts.total > bestScore {
			position += 1
			bestScore = ts.total
		}
		var prefix string
		if position == 1 {
			prefix = ":first_place:"
		} else if position == 2 {
			prefix = ":second_place:"
		} else if position == 3 {
			prefix = ":third_place:"
		} else {
			prefix = fmt.Sprintf("%d", position)
		}
		if len(prefix) == 1 {
			prefix += " "
		}

		// TODO: do this in more robust way to prevent OOB
		sb.WriteString(fmt.Sprintf("%s| <@%s> **%d moves**:  %d  %d  %d\n", prefix, ts.userID, ts.total, ts.scores[0], ts.scores[1], ts.scores[2]))
	}

	// print leaderboard
	_, err := dg.ChannelMessageSend(instance.channelID, sb.String())
	if err != nil {
		return fmt.Errorf("printing leaderboard: %v", err)
	}
	return nil
}

func cancelTournament(dg *discordgo.Session, instance *discordInstance, t *tournament) error {
	instance.activeTournament = nil

	content := ":x: tournament cancelled due to error, please try again later"
	_, err := dg.ChannelMessageSend(instance.channelID, content)
	if err != nil {
		return fmt.Errorf("cancelling tournament: %v", err)
	}
	return nil
}

func tournamentPuzzleContent(member *discordgo.Member, tg tournamentGame, endTime time.Time) string {

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("`-------------------------------------------------`\n"))
	sb.WriteString(fmt.Sprintf("**Tournament Puzzle #%d** -- ", tg.index+1))
	sb.WriteString(fmt.Sprintf("Time Remaining: **<t:%d:R>**", endTime.Unix()))

	return sb.String()
}
