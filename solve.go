package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func (s *server) handleSolve(dg *discordgo.Session, i *discordgo.InteractionCreate) error {
	err := dg.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: 1 << 6, // ephemeral
		},
	})
	if err != nil {
		return fmt.Errorf("responding solve ack: %v", err)
	}

	// parse user moves
	if len(i.Interaction.ApplicationCommandData().Options) == 0 {
		return fmt.Errorf("No moves provided to /solve: %v", i)
	}
	moveStr := i.Interaction.ApplicationCommandData().Options[0].Value
	moves, err := parseMoves(moveStr.(string))
	if err != nil {
		content := fmt.Sprintf("'%s' is not a valid move format. Please see **/help**", moveStr)
		_, err = dg.InteractionResponseEdit(i.Interaction,
			&discordgo.WebhookEdit{
				Content: &content,
			},
		)
		if err != nil {
			return fmt.Errorf("Failed to respond with invalid moves: %v", err)
		}
		return nil
	}

	// look up instance
	instance := s.instances[i.GuildID]

	// TODO: think if this wouldn't be better as an entirely separate command
	// if there is an included ID, try to hydrate the provided puzzle
	if len(i.Interaction.ApplicationCommandData().Options) == 2 {
		puzzleID := i.Interaction.ApplicationCommandData().Options[1].Value.(string)
		puzzleID = strings.TrimSpace(puzzleID)
		// sanity check, if they provided the ID of the currently active puzzle. Take normal codepath
		// otherwise handler specifically for old puzzles
		if instance.activeGame == nil || instance.activeGame.id != puzzleID {
			return solveEncodedPuzzle(dg, i, puzzleID, moves, moveStr.(string), instance)
		}

	}

	// no active puzzle to solve
	if instance.activeGame == nil {
		content := fmt.Sprintf("There is no active puzzle. Please use **/puzzle** to create one")
		_, err = dg.InteractionResponseEdit(i.Interaction,
			&discordgo.WebhookEdit{
				Content: &content,
			},
		)
		return nil
	}

	// validate solution
	success := validate(instance.activeGame, instance.activeGame.board, moves, instance.activeGame.activeGoal)
	var content string
	if success {

		content = fmt.Sprintf(":white_check_mark: Puzzle Solved: %s", moveStr)
		_, err = dg.InteractionResponseEdit(i.Interaction,
			&discordgo.WebhookEdit{
				Content: &content,
			},
		)

		if i.Interaction.Member == nil {
			return fmt.Errorf("User invoked solve in a DM? how did this happen?")
		}

		// extra stuff if on arena server
		if i.Interaction.GuildID == ArenaServerID {
			if err := arenaSolution(dg, i.Interaction, instance, s.db, moves); err != nil {
				log.Printf("processing arena solution: %v", err)
				return fmt.Errorf("processing arena solution: %v", err)
			}
		} else {

			solutions := instance.getSolutions(instance.activeGame.id)
			bestForUser := len(solutions.get(i.Interaction.Member.User.ID))
			if bestForUser == 0 {
				bestForUser = 999
			}
			if len(moves) < bestForUser {
				solutions.set(i.Interaction.Member.User.ID, moves)
			}

			// only print solution info if there is not an active tournament
			if instance.activeTournament == nil {
				var content string
				if len(moves) == instance.activeGame.lenOptimalSolution {
					content = fmt.Sprintf("<@%s> solved with an :tada:**optimal**:tada: %d move solution", i.Interaction.Member.User.ID, len(moves))
				} else {
					content = fmt.Sprintf("<@%s> solved with a %d move solution", i.Interaction.Member.User.ID, len(moves))
				}
				dg.ChannelMessageSend(i.Interaction.ChannelID, content)
			}
		}

	} else {
		content = fmt.Sprintf(":x: %s is not a valid solution to this puzzle", moveStr)
		_, err = dg.InteractionResponseEdit(i.Interaction,
			&discordgo.WebhookEdit{
				Content: &content,
			},
		)
	}
	if err != nil {
		log.Printf("unable to print puzzle: %v\n", err)
		return fmt.Errorf("Editing response with puzzle: %v", err)
	}

	//spew.Dump(i.ApplicationCommandData())
	return nil
}

// TODO: clean up params
func solveEncodedPuzzle(dg *discordgo.Session, i *discordgo.InteractionCreate, puzzleID string, moves []move, moveStr string, instance *discordInstance) error {

	decodedGame, err := decode(strings.TrimPrefix(puzzleID, "#"))
	if err != nil {
		content := fmt.Sprintf("Invalid puzzle_id. If you are solving the active puzzle, leave this option blank")
		_, err = dg.InteractionResponseEdit(i.Interaction,
			&discordgo.WebhookEdit{
				Content: &content,
			},
		)
		return nil
	}

	success := validate(decodedGame, decodedGame.board, moves, decodedGame.activeGoal)
	var content string
	if success {

		content = fmt.Sprintf(":white_check_mark: Puzzle Solved: %s", moveStr)
		_, err = dg.InteractionResponseEdit(i.Interaction,
			&discordgo.WebhookEdit{
				Content: &content,
			},
		)

		if i.Interaction.Member == nil {
			return fmt.Errorf("User invoked solve in a DM? how did this happen?")
		}

		solutions := instance.getSolutions(puzzleID)
		bestForUser := len(solutions.get(i.Interaction.Member.User.ID))
		if bestForUser == 0 {
			bestForUser = 999
		}
		if len(moves) < bestForUser {
			solutions.set(i.Interaction.Member.User.ID, moves)
		}

		var content string
		content = fmt.Sprintf("<@%s> solved non-active puzzle #**%s** with a %d move solution. (optimal moves not calculated for old puzzles)", i.Interaction.Member.User.ID, puzzleID, len(moves))
		dg.ChannelMessageSend(i.Interaction.ChannelID, content)

	} else {
		content = fmt.Sprintf(":x: %s is not a valid solution to puzzle %s", moveStr, puzzleID)
		_, err = dg.InteractionResponseEdit(i.Interaction,
			&discordgo.WebhookEdit{
				Content: &content,
			},
		)
	}
	if err != nil {
		log.Printf("unable to print puzzle: %v\n", err)
		return fmt.Errorf("Editing response with puzzle: %v", err)
	}

	return nil

}
