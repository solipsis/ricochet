package main

import (
	"bytes"
	"fmt"
	"image/png"
	"log"

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

	//	content := printBoard(g.board, g.size, g.robots, g.activeGoal)
	file := &discordgo.File{
		Name:        "board.png",
		ContentType: "image/png",
		Reader:      &buf,
	}

	dg.ChannelMessageSendComplex(instance.channelID, &discordgo.MessageSend{
		Content: "Puzzle #X",
		Files:   []*discordgo.File{file},
	})

	content := "Puzzle created successfully"
	_, err = dg.InteractionResponseEdit(i.Interaction,
		&discordgo.WebhookEdit{
			Content: &content,
			//			Components: &[]discordgo.MessageComponent{},
		},
	)
	if err != nil {
		log.Printf("unable to print puzzle: %v\n", err)
		return fmt.Errorf("Editing response with puzzle: %v", err)
	}

	return nil
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
