package main

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

func handlePuzzle(dg *discordgo.Session, i *discordgo.InteractionCreate) error {

	err := dg.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		//	Data: &discordgo.InteractionResponseData{
		//	Flags: 1 << 6, // ephemeral
		//	},
	})
	if err != nil {
		log.Printf("repsonding with deferred ack: %v", err)
		return fmt.Errorf("repsonding with deferred ack: %v", err)
	}

	g := randomGame()
	g.optimalMoves = g.preCompute(g.activeGoal.position)

	content := printBoard(g.board, g.size, g.robots, g.activeGoal)
	_, err = dg.InteractionResponseEdit(i.Interaction,
		&discordgo.WebhookEdit{
			Content:    &content,
			Components: &[]discordgo.MessageComponent{},
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
