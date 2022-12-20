package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func (s *server) handleShare(dg *discordgo.Session, i *discordgo.InteractionCreate) error {
	// ack
	err := dg.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: 1 << 6, // ephemeral
		},
	})
	if err != nil {
		log.Printf("responding help ack: %v\n", err)
		return fmt.Errorf("respoding help ack: %v", err)
	}

	// look up instance
	instance := s.instances[i.GuildID]
	game := instance.activeGame

	if len(i.Interaction.ApplicationCommandData().Options) == 1 {
		puzzleID := i.Interaction.ApplicationCommandData().Options[0].Value.(string)
		puzzleID = strings.TrimSpace(puzzleID)

		// sanity check, if they provided the ID of the currently active puzzle. Take normal codepath
		// otherwise handler specifically for old puzzles
		if instance.activeGame == nil || instance.activeGame.id != puzzleID {
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
			game = decodedGame
		}
	}

	// no puzzle active
	if game == nil {
		content := "There is no active puzzle"
		_, err = dg.InteractionResponseEdit(i.Interaction,
			&discordgo.WebhookEdit{
				Content: &content,
			},
		)
		if err != nil {
			return fmt.Errorf("sending no active puzzle response: %v", err)
		}
		return nil
	}

	currentMoves := instance.getSolutions(game.id).get(i.Member.User.ID)
	if len(currentMoves) == 0 {
		content := "You have not solved this puzzle"
		_, err = dg.InteractionResponseEdit(i.Interaction,
			&discordgo.WebhookEdit{
				Content: &content,
			},
		)
		if err != nil {
			return fmt.Errorf("sending help response: %v", err)
		}
		return nil
	}

	// build answer string
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<@%s> used **/share** for puzzle: #%s\n||", i.Member.User.ID, game.id))
	for idx, m := range currentMoves {
		switch m.id {
		case 'R':
			sb.WriteString(":red_circle:")
		case 'B':
			sb.WriteString(":blue_circle:")
		case 'G':
			sb.WriteString(":green_circle:")
		case 'Y':
			sb.WriteString(":yellow_circle:")
		}
		switch m.dir {
		case UP:
			sb.WriteString(":arrow_up:")
		case DOWN:
			sb.WriteString(":arrow_down:")
		case LEFT:
			sb.WriteString(":arrow_left:")
		case RIGHT:
			sb.WriteString(":arrow_right:")
		}

		if idx != len(currentMoves)-1 {
			sb.WriteString(" - ")
		}
	}
	sb.WriteString("||")

	// render solution to gif form
	gif, err := renderGif(game, currentMoves)
	if err != nil {
		return fmt.Errorf("rendering solution gif: %v", err)
	}
	file := &discordgo.File{
		Name:        "SPOILER_solution.gif", // spoiler prefix required
		ContentType: "image/gif",
		Reader:      &gif,
	}

	_, err = dg.ChannelMessageSendComplex(instance.channelID, &discordgo.MessageSend{
		Content: sb.String(),
		Files:   []*discordgo.File{file},
	})
	if err != nil {
		return fmt.Errorf("sending share string: %v", err)
	}

	// edit original response
	content := "Solution shared"
	_, err = dg.InteractionResponseEdit(i.Interaction,
		&discordgo.WebhookEdit{
			Content: &content,
		},
	)
	if err != nil {
		return fmt.Errorf("editing original share response: %v", err)
	}

	return nil
}
