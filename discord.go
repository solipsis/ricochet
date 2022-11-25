package main

import (
	"bytes"
	"fmt"
	"image/png"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func findChannel(dg *discordgo.Session, guildID string) (*discordgo.Channel, error) {
	// check if channel already exists
	channels, err := dg.GuildChannels(guildID)
	if err != nil {
		return nil, fmt.Errorf("Reading existing channels: %v", err)
	}
	for _, ch := range channels {

		if ch.Type != discordgo.ChannelTypeGuildText {
			continue
		}
		if ch.Name == "ricochet" {
			return ch, nil
		}
	}

	return nil, nil
}

func (s *server) handleSolve(dg *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

func (s *server) handleHelp(dg *discordgo.Session, i *discordgo.InteractionCreate) error {
	// ack
	err := dg.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: 1 << 6, // ephemeral
		},
	})

	// respond
	var sb strings.Builder
	sb.WriteString("**Commands**:\n")
	sb.WriteString("  **/puzzle**: Generate a new puzzle to solve\n")
	sb.WriteString("  **/solve**: Submit a solution to the current puzzle\n")
	sb.WriteString("\n**How to Play**:\n")
	sb.WriteString("1. robots may only move Up, Down, Left, or Right\n")
	sb.WriteString("2. robots move in a straight line until hitting a wall or another robot\n")
	sb.WriteString("3. you can move robots in any order and as many times as you like\n")
	sb.WriteString("\nSubmit your answer like \"**/solve RU-GD-BL-YR**\"\n")
	sb.WriteString("R=red, G=green, B=blue, Y=yellow\n")
	sb.WriteString("U=up, D=down, L=left, R=right\n")

	content := sb.String()
	_, err = dg.InteractionResponseEdit(i.Interaction,
		&discordgo.WebhookEdit{
			Content: &content,
		},
	)
	if err != nil {
		log.Printf("sending help response: %v\n", err)
		return fmt.Errorf("sending help response: %v", err)
	}
	return nil
}

func (s *server) handlePuzzle(dg *discordgo.Session, i *discordgo.InteractionCreate) error {

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

	// look up instance
	instance := s.instances[i.GuildID]

	/*
		g := randomGame()
		g.optimalMoves = g.preCompute(g.activeGoal.position)
	*/
	g := <-s.categorizer.medium
	instance.activeGame = g

	img, err := render(g)
	if err != nil {
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

	dg.ChannelMessageSendComplex(instance.channelID, &discordgo.MessageSend{
		Content: puzzleContent(instance.puzzleIdx, instance.activeGame),
		Files:   []*discordgo.File{file},
	})

	content := "Puzzle created successfully"
	_, err = dg.InteractionResponseEdit(i.Interaction,
		&discordgo.WebhookEdit{
			Content: &content,
		},
	)
	if err != nil {
		log.Printf("unable to print puzzle: %v\n", err)
		return fmt.Errorf("Editing response with puzzle: %v", err)
	}

	if !s.isSearching {
		go lookForSolutions(s)
	}

	return nil
}

func puzzleContent(num int, g *game) string {

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("**Puzzle #%d**\n", num))

	var color string
	switch g.activeGoal.id {
	case 'R':
		color = "Red"
	case 'B':
		color = "Blue"
	case 'Y':
		color = "Yellow"
	case 'G':
		color = "Green"
	}
	colorEmoji := fmt.Sprintf(":%s_square:", strings.ToLower(color))

	sb.WriteString(fmt.Sprintf("Get the %s robot to the goal %s", color, colorEmoji))

	return sb.String()
}

var slashCommands = []*discordgo.ApplicationCommand{
	{
		Name:        "puzzle",
		Description: "generate a new puzzle",
	},
	{
		Name:        "submit",
		Description: "submit an answer to the current puzzle",
	},
	{
		Name:        "help",
		Description: "how to interact with ricochet-robotbot",
	},
}

// registerCommands fully refreshes the slashCommand list for the provided guild
func registerCommands(dg *discordgo.Session, guildID string) error {

	// overwrite old commands and update new commands
	_, err := dg.ApplicationCommandBulkOverwrite(DiscordApplicationID, guildID, slashCommands)
	if err != nil {
		log.Printf("Registering application commands: %v, for guild: %s", err, guildID)
	}

	return nil
}
