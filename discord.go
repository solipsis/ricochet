package main

import (
	"bytes"
	"fmt"
	"image/png"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

const TestServerID = "810570434453438475"  // my discord
const ArenaServerID = "692911659169218560" // arena
//const ArenaServerID = "8105704344534384751" //not my discord

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
		if ch.Name == "ricochet" || ch.Name == "🤖｜ricochet" {
			return ch, nil
		}
	}

	return nil, nil
}

func (s *server) handleHowToPlay(dg *discordgo.Session, i *discordgo.InteractionCreate) error {
	return s.handleHelp(dg, i)
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
	sb.WriteString("**Ricochet-Robotbot** v0.0.3\n")
	sb.WriteString("----------------------------\n\n")
	sb.WriteString("**Commands**:\n")
	sb.WriteString("  **/puzzle**: Generate a new puzzle to solve\n")
	sb.WriteString("  **/solve**: Submit a solution to the current puzzle\n")
	sb.WriteString("  **/share**: Share your solution to the current puzzle\n")
	sb.WriteString("  **/how-to-play**: Additional explanation of game rules\n")
	sb.WriteString("  **/tournament**: Start a 3 puzzle timed tournament\n")
	sb.WriteString("\n**Coming Soon**:\n")
	sb.WriteString("- Load specific puzzles\n")
	sb.WriteString("- More Boards\n")
	sb.WriteString("- Rules Variants\n")
	sb.WriteString("\n**How to Play**:\n")
	sb.WriteString("1. robots may only move Up, Down, Left, or Right\n")
	sb.WriteString("2. robots move in a straight line until hitting a wall or another robot\n")
	sb.WriteString("3. you can move robots in any order and as many times as you like\n")
	sb.WriteString("\nSubmit your answer using the **/solve** command e.g. \"**/solve RU-GD-BL-YR**\"\n")
	sb.WriteString("R=red, G=green, B=blue, Y=yellow\n")
	sb.WriteString("U=up, D=down, L=left, R=right\n")
	sb.WriteString("\n**Example Game**:\n")
	sb.WriteString("The attached images show the user creating a new puzzle with the **/puzzle** command. Then solving it with the **/solve** command. Then sharing their answer with the **/share** command")

	// load sample images
	images := []*discordgo.File{}
	for _, fname := range []string{"tut1.png", "tut2.png", "tut3.png"} {
		buf, _ := ioutil.ReadFile(filepath.Join("ricochet-images", fname))
		img := bytes.NewBuffer(buf)
		images = append(images, &discordgo.File{
			Name:        fname,
			ContentType: "image/png",
			Reader:      img,
		})
	}

	content := sb.String()
	_, err = dg.InteractionResponseEdit(i.Interaction,
		&discordgo.WebhookEdit{
			Content: &content,
			Files:   images,
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
	if instance.activeGame != nil {
		//if instance.solutionTracker.numSubmitted() == 0
		optimalFound := len(instance.getSolutions(instance.activeGame.id).currentBest()) == instance.activeGame.lenOptimalSolution
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

	// tournament already in progress
	if instance.activeTournament != nil {
		content := "There is currently a tournament in progress. You may only request a puzzle after it is over"
		_, err := dg.InteractionResponseEdit(i.Interaction,
			&discordgo.WebhookEdit{
				Content: &content,
			},
		)
		return err
	}

	var g *game
	if len(i.Interaction.ApplicationCommandData().Options) == 0 {
		g = <-s.categorizer.medium
		g.difficulty = MEDIUM
	} else if i.Interaction.ApplicationCommandData().Options[0].Value == "easy" {
		g = <-s.categorizer.easy
		g.difficulty = EASY
	} else if i.Interaction.ApplicationCommandData().Options[0].Value == "medium" {
		g = <-s.categorizer.medium
		g.difficulty = MEDIUM
	} else if i.Interaction.ApplicationCommandData().Options[0].Value == "hard" {
		g = <-s.categorizer.hard
		g.difficulty = HARD
	} else {
		g = <-s.categorizer.medium
		g.difficulty = MEDIUM
	}

	if !s.isSearching {
		go lookForSolutions(s)
	}

	instance.activeGame = g
	//instance.solution = &solutionTracker{}
	instance.puzzleTimestamp = time.Now()

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

	_, err = dg.ChannelMessageSendComplex(instance.channelID, &discordgo.MessageSend{
		Content: puzzleContent(i.Interaction.Member, instance.activeGame),
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

	content := "Puzzle created successfully"
	_, err = dg.InteractionResponseEdit(i.Interaction,
		&discordgo.WebhookEdit{
			Content: &content,
		},
	)
	if err != nil {
		log.Printf("unable to respond to puzzle creation request: %v\n", err)
		return fmt.Errorf("Editing response with puzzle: %v", err)
	}

	return nil
}

func puzzleContent(member *discordgo.Member, g *game) string {

	var displayName string
	if member.Nick != "" {
		displayName = member.Nick
	} else {
		displayName = member.User.Username
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s used **/puzzle**\n", displayName))
	sb.WriteString(fmt.Sprintf("**Puzzle:** #__%s__ -- %s\n", g.id, g.difficulty))

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
		Description: "generate a new puzzle. This will overwrite the current active puzzle",
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
			{
				Name:        "puzzle_id",
				Description: "submit an answer for a puzzle. If left blank it will submit for the currently active puzzle",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    false,
			},
		},
	},
	{
		Name:        "help",
		Description: "how to interact with ricochet-robotbot",
	},
	{
		Name:        "share",
		Description: "share your solution to the current problem",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "puzzle_id",
				Description: "share your solution for a puzzle. If blank it will share your solution to the current puzzle",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    false,
			},
		},
	},
	{
		Name:        "how-to-play",
		Description: "short tutorial on playing the game",
	},
	{
		Name:        "tournament",
		Description: "start a 3 puzzle timed tournament",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "difficulty",
				Description: "difficulty of tournament puzzles",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
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
			{
				Name:        "duration",
				Description: "how many minutes per puzzle (Min: 1, Max: 10)",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
				MaxValue:    10,
			},
		},
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
