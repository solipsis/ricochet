package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

const DiscordApplicationID = "1044049636106706974"
const gameBuffer = 5

type server struct {
	categorizer *categorizer
	isSearching bool
	instances   map[string]*discordInstance
}

type discordInstance struct {
	serverID   string
	channelID  string
	activeGame *game
}

func (s *server) run() {
	s.instances = make(map[string]*discordInstance)

	discordToken := os.Getenv("RICOCHET_DISCORD_TOKEN")
	if discordToken == "" {
		log.Fatal("Missing required env: RICOCHET_DISCORD_TOKEN")
	}

	dg, err := discordgo.New("Bot " + discordToken)
	if err != nil {
		log.Fatalf("failed to authenticate with discord: %v", err)
	}
	fmt.Println(dg)

	// Handler that will register all known slash commands whenever the bot is invited
	// to a new guild or restarted.
	dg.AddHandler(func(dg *discordgo.Session, gc *discordgo.GuildCreate) {
		log.Println("Invited to guild:", gc.Name)

		// look for a ricochet channel
		channel, err := findChannel(dg, gc.ID)
		if err != nil {
			log.Printf("unable to find channels: %v", err)
			return
		}

		// TODO: Create if not exists

		// local dev reset on start
		/*
			channelID := "1044057773991800882"
			oldMessages, err := dg.ChannelMessages(channelID, 100, "", "", "")
			if err != nil {
				log.Printf("Unable to delete existing messages: %v\n", err)
			}
			for _, msg := range oldMessages {
				if err := dg.ChannelMessageDelete(channelID, msg.ID); err != nil {
					log.Printf("Unable to delete message: %v\n", err)
				}
			}
		*/

		if err := registerCommands(dg, gc.Guild.ID); err != nil {
			log.Fatalf("Unable to update commands: %v\n", err)
		}

		s.instances[gc.Guild.ID] = &discordInstance{
			serverID:  gc.Guild.ID,
			channelID: channel.ID,
		}
	})

	// top level handler for slash commands, user commands, and continued interactions
	dg.AddHandler(func(dg *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
		case discordgo.InteractionMessageComponent:
			fmt.Println("Interaction Continue")
			//handleInteractionContinue(dg, i)
		case discordgo.InteractionApplicationCommand:
			switch i.ApplicationCommandData().Name {
			case "puzzle":
				err := s.handlePuzzle(dg, i)
				if err != nil {
					log.Printf("puzzle handler: %v", err)
				}
			case "solve":
				err := s.handleSolve(dg, i)
				if err != nil {
					log.Printf("solve handler: %v", err)
				}
			default:
				log.Println("Unknown Command:", i.ApplicationCommandData().Name)
			}
		case discordgo.InteractionModalSubmit:
			fmt.Println("modal")
		}

	})

	if err := dg.Open(); err != nil {
		log.Fatalf("opening discord connection: %v\n", err)
	}

	cat := categorizer{
		easy:   make(chan (*game), gameBuffer),
		medium: make(chan (*game), gameBuffer),
		hard:   make(chan (*game), gameBuffer),
	}
	s.categorizer = &cat

	lookForSolutions(s)

	fmt.Println("infinite loop")
	for {
		time.Sleep(10 * time.Second)

	}

	// listen for discord events
}

func lookForSolutions(s *server) {
	s.isSearching = true // not thread-safe but probably will never matter
	var wg sync.WaitGroup
	for x := 0; x < 4; x++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				if len(s.categorizer.easy) == gameBuffer && len(s.categorizer.medium) == gameBuffer && len(s.categorizer.hard) == gameBuffer {
					break
				}

				rg := randomGame()
				rg.optimalMoves = rg.preCompute(rg.activeGoal.position)
				res := rg.solve(18)
				moves, _ := parseMoves(res)
				numMoves := len(moves)

				// Add solution to proper buffer. Discard if that buffer already has enough solutions
				// of that length
				if numMoves >= 5 && numMoves <= 7 {
					select {
					case s.categorizer.easy <- rg:
						fmt.Println("Easy found:")
					default:
						//			fmt.Println("discarding easy")
					}
				} else if numMoves >= 8 && numMoves <= 12 {
					select {
					case s.categorizer.medium <- rg:
						fmt.Println("Medium found:")
					default:
						//			fmt.Println("discarding medium")
					}
				} else if numMoves >= 13 && numMoves <= 20 {
					select {
					case s.categorizer.hard <- rg:
						fmt.Println("Hard found:", numMoves)
					default:
						//			fmt.Println("discarding hard")
					}
				}

			}
		}()
	}
	wg.Wait()
	s.isSearching = false

}

func (s *server) servePuzzle(difficulty string) *game {
	var g *game
	switch difficulty {
	case "easy":
		g = <-s.categorizer.easy
	case "medium":
		g = <-s.categorizer.medium
	case "hard":
		g = <-s.categorizer.hard
	}
	return g
}
