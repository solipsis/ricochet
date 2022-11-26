package main

import (
	"bytes"
	"fmt"
	"image/png"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

const ArenaServerID = "810570434453438475" // my discord

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

		if i.Interaction.Member != nil {

			// extra stuff if on arena server
			if i.Interaction.GuildID == ArenaServerID {
				arenaSolution(dg, i.Interaction, instance, moves)
			} else {

				bestForUser := len(instance.solutionTracker.get(i.Interaction.Member.User.ID))
				if bestForUser == 0 {
					bestForUser = 999
				}
				if len(moves) < bestForUser {
					instance.solutionTracker.set(i.Interaction.Member.User.ID, moves)
				}

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

func (s *server) handleHelp(dg *discordgo.Session, i *discordgo.InteractionCreate) error {
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

	// haven't solved current puzzle
	if instance.activeGame != nil && instance.solutionTracker.numSubmitted() == 0 {
		content := "Current puzzle must be solved before requesting a new one"
		dg.InteractionResponseEdit(i.Interaction,
			&discordgo.WebhookEdit{
				Content: &content,
			},
		)
		return nil
	}

	// grab a game of the proper difficulty
	var g *game
	if len(i.Interaction.ApplicationCommandData().Options) == 0 {
		g = <-s.categorizer.medium
		g.difficultyName = "medium"
	} else if i.Interaction.ApplicationCommandData().Options[0].Value == "easy" {
		g = <-s.categorizer.easy
		g.difficultyName = "easy"
	} else if i.Interaction.ApplicationCommandData().Options[0].Value == "medium" {
		g = <-s.categorizer.medium
		g.difficultyName = "medium"
	} else if i.Interaction.ApplicationCommandData().Options[0].Value == "hard" {
		g = <-s.categorizer.hard
		g.difficultyName = "hard"
	} else {
		g = <-s.categorizer.medium
		g.difficultyName = "medium"
	}

	instance.activeGame = g
	instance.solutionTracker = &solutionTracker{}

	var moveStrs []string
	for _, m := range g.moves {
		moveStrs = append(moveStrs, m.String())
	}
	fmt.Println("Optimal:", strings.Join(moveStrs, "-"))

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
	sb.WriteString(fmt.Sprintf("**Puzzle #%d** -- %s\n", num, g.difficultyName))

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
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "difficulty",
				Description: "easy, medium, or hard",
				Type:        discordgo.ApplicationCommandOptionString,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{
						Name:  "easy",
						Value: "easy",
					},
					{
						Name:  "medium",
						Value: "medium",
					},
					{
						Name:  "hard",
						Value: "hard",
					},
				},
			},
		},
	},
	{
		Name:        "solve",
		Description: "submit an answer to the current puzzle",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "moves",
				Description: "List of moves i.e. RU-GD-BR-YL",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
			},
		},
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
