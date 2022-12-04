package main

import (
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/image/draw"
)

func render(g *game) (image.Image, error) {

	dst := image.NewNRGBA(image.Rect(0, 0, 16*16, 16*16))
	// one row at a time
	for row := 0; row < g.size; row += 1 {
		for col := 0; col < g.size; col += 1 {
			/*
				f, err := os.Open(filepath.Join("ricochet-images", "vanilla3.png"))
				if err != nil {
					log.Fatalf("opnening tile: %v", err)
				}
				tile, err := png.Decode(f)
				if err != nil {
					log.Fatalf("decoding tile: %v", err)
				}
			*/
			sq := g.board[row*g.size+col]
			tile := pickTile(sq)

			x := (col * 16)
			y := (row * 16)
			draw.BiLinear.Scale(dst, image.Rect(x, y, x+16, y+16), tile, tile.Bounds(), draw.Over, nil)

			if sq&square(ROBOT) != 0 {
				draw.BiLinear.Scale(dst, image.Rect(x, y, x+16, y+16), tile, tile.Bounds(), draw.Over, nil)
			}
		}
	}

	// draw robots
	for _, r := range g.robots {
		robotImg := pickRobot(*r)
		row := r.position / 16
		col := r.position % 16

		x := int(col * 16)
		y := int(row * 16)
		draw.BiLinear.Scale(dst, image.Rect(x, y, x+16, y+16), robotImg, robotImg.Bounds(), draw.Over, nil)
	}

	// draw goal
	goalImg := pickGoal(g.activeGoal)
	row := g.activeGoal.position / 16
	col := g.activeGoal.position % 16

	x := int(col * 16)
	y := int(row * 16)
	draw.BiLinear.Scale(dst, image.Rect(x, y, x+16, y+16), goalImg, goalImg.Bounds(), draw.Over, nil)
	//return dst, nil

	upscaled := image.NewNRGBA(image.Rect(0, 0, 32*32, 32*32))
	draw.NearestNeighbor.Scale(upscaled, upscaled.Bounds(), dst, dst.Bounds(), draw.Over, nil)

	return upscaled, nil

	/*
		out, err := os.Create("out.png")
		if err != nil {
			log.Fatalf("creating output: %v", err)
		}
		if err := png.Encode(out, upscaled); err != nil {
			log.Fatalf("Encoding: %v", err)
		}
	*/
}

func pickTile(sq square) image.Image {

	s := direction(sq)
	var fname string
	switch {
	case s&UP != 0 && s&LEFT != 0:
		fname = "up-left3.png"
	case s&UP != 0 && s&RIGHT != 0:
		fname = "up-right3.png"
	case s&DOWN != 0 && s&LEFT != 0:
		fname = "down-left3.png"
	case s&DOWN != 0 && s&RIGHT != 0:
		fname = "down-right3.png"
	case s&LEFT != 0 && s&RIGHT != 0:
		fname = "left-right3.png"
	case s&UP != 0 && s&DOWN != 0:
		fname = "up-down3.png"
	case s&UP != 0:
		fname = "up3.png"
	case s&DOWN != 0:
		fname = "down3.png"
	case s&LEFT != 0:
		fname = "left3.png"
	case s&RIGHT != 0:
		fname = "right3.png"
	default:
		fname = "vanilla3.png"
	}

	f, err := os.Open(filepath.Join("ricochet-images", fname))
	if err != nil {
		log.Fatalf("opnening tile: %v", err)
	}
	tile, err := png.Decode(f)
	if err != nil {
		log.Fatalf("decoding tile: %v", err)
	}
	return tile
}

func pickRobot(r robot) image.Image {

	var fname string
	switch r.id {
	case 'R':
		fname = "robot-red.png"
	case 'B':
		fname = "robot-blue.png"
	case 'G':
		fname = "robot-green.png"
	case 'Y':
		fname = "robot-yellow.png"
	}

	f, err := os.Open(filepath.Join("ricochet-images", fname))
	if err != nil {
		log.Fatalf("opnening tile: %v", err)
	}
	tile, err := png.Decode(f)
	if err != nil {
		log.Fatalf("decoding tile: %v", err)
	}
	return tile
}

func pickGoal(g Goal) image.Image {
	fmt.Printf("goal: %d %c\n", g, g)

	var fname string
	switch g.id {
	case 'R':
		fname = "goal-red2.png"
	case 'B':
		fname = "goal-blue2.png"
	case 'G':
		fname = "goal-green2.png"
	case 'Y':
		fname = "goal-yellow2.png"
	}

	f, err := os.Open(filepath.Join("ricochet-images", fname))
	if err != nil {
		log.Fatalf("opnening tile: %v", err)
	}
	tile, err := png.Decode(f)
	if err != nil {
		log.Fatalf("decoding tile: %v", err)
	}
	return tile
}
