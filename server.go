package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v4/pgxpool"
)

const DiscordApplicationID = "1044049636106706974" // PROD
//const DiscordApplicationID = "1047352593430626305" // DEV
const gameBuffer = 5

type server struct {
	categorizer *categorizer
	isSearching bool
	instances   map[string]*discordInstance
	db          *pgxpool.Pool
}

type discordInstance struct {
	serverID         string
	channelID        string
	puzzleIdx        int
	activeGame       *game
	activeTournament *tournament

	puzzleTimestamp time.Time

	// TODO: this grows unbounded. Need to remove entries at some point
	solutions    map[string]*solutionTracker
	solutionLock sync.RWMutex
}

func (di *discordInstance) getSolutions(id string) *solutionTracker {
	di.solutionLock.Lock()
	defer di.solutionLock.Unlock()
	if di.solutions == nil {
		di.solutions = make(map[string]*solutionTracker)
	}

	tracker := di.solutions[id]
	if tracker == nil {
		di.solutions[id] = &solutionTracker{}
	}

	return di.solutions[id]
}

type solutionTracker struct {
	lock               sync.Mutex
	submittedSolutions map[string][]move
}

func (st *solutionTracker) set(key string, moves []move) {
	st.lock.Lock()
	defer st.lock.Unlock()
	if st.submittedSolutions == nil {
		st.submittedSolutions = make(map[string][]move)
	}
	st.submittedSolutions[key] = moves
}

func (st *solutionTracker) get(key string) []move {
	st.lock.Lock()
	defer st.lock.Unlock()
	return st.submittedSolutions[key]
}

func (st *solutionTracker) users() []string {
	st.lock.Lock()
	defer st.lock.Unlock()

	var users []string
	//fmt.Printf("users() submittedSolutions: %+v\n", st.submittedSolutions)
	for k, _ := range st.submittedSolutions {
		users = append(users, k)
	}
	return users
}

func (st *solutionTracker) currentBest() []move {
	st.lock.Lock()
	defer st.lock.Unlock()

	bestNum := 999
	var bestMoves []move
	for _, v := range st.submittedSolutions {
		if len(v) < bestNum {
			bestNum = len(v)
			bestMoves = v
		}
	}

	return bestMoves
}

func (st *solutionTracker) numSubmitted() int {
	st.lock.Lock()
	defer st.lock.Unlock()
	return len(st.submittedSolutions)
}

func (s *server) run() {
	s.instances = make(map[string]*discordInstance)

	discordToken := os.Getenv("RICOCHET_DISCORD_TOKEN") // PROD
	//discordToken := os.Getenv("RICOCHET_DEV_DISCORD_TOKEN") //dev
	if discordToken == "" {
		log.Fatal("Missing required env: RICOCHET_DISCORD_TOKEN")
	}

	dg, err := discordgo.New("Bot " + discordToken)
	if err != nil {
		log.Fatalf("failed to authenticate with discord: %v", err)
	}

	// connect to db
	dbURL := os.Getenv("DATABASE_URL")
	conn, err := pgxpool.Connect(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("unable to connect to db: %v", err)
	}
	defer conn.Close()
	s.db = conn

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
		if channel == nil {
			//channel, err = dg.GuildChannelCreate(gc.ID, "ricochet", discordgo.ChannelTypeGuildText)

			channel, err = dg.GuildChannelCreateComplex(gc.ID, discordgo.GuildChannelCreateData{
				Name:  "ricochet",
				Type:  discordgo.ChannelTypeGuildText,
				Topic: "**/how-to-play** to get started",
			})
			if err != nil {
				log.Printf("unable to create ricochet channel")
				return
			}
		}

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
			log.Printf("Unable to update commands: %v\n", err)
			return
		}

		s.instances[gc.Guild.ID] = &discordInstance{
			serverID:  gc.Guild.ID,
			channelID: channel.ID,
		}

		//		var sb strings.Builder
		//		sb.WriteString("**Ricochet-Robotbot** v0.0.1\n")
		//		sb.WriteString("\ntype **/help** to get started\n")
		//		if _, err := dg.ChannelMessageSend(channel.ID, sb.String()); err != nil {
		//			log.Printf("unable to post intro message: %v", err)
		//		}

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
			case "help":
				err := s.handleHelp(dg, i)
				if err != nil {
					log.Printf("help handler: %v", err)
				}
			case "share":
				err := s.handleShare(dg, i)
				if err != nil {
					log.Printf("share handler: %v", err)
				}
			case "how-to-play":
				err := s.handleHowToPlay(dg, i)
				if err != nil {
					log.Printf("how-to-play handler: %v", err)
				}
			case "tournament":
				err := s.handleTournament(dg, i)
				if err != nil {
					log.Printf("tournament handler: %v", err)
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
		easy:    make(chan (*game), gameBuffer),
		medium:  make(chan (*game), gameBuffer),
		hard:    make(chan (*game), gameBuffer),
		extreme: make(chan (*game), gameBuffer),
	}
	s.categorizer = &cat

	lookForSolutions(s)

	fmt.Println("infinite loop")
	for {
		time.Sleep(10 * time.Second)

	}

	// listen for discord events
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
	case "extreme":
		g = <-s.categorizer.extreme
	}
	return g
}
