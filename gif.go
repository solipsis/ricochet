package main

import (
	"bytes"
	"image"
	"image/color/palette"
	"image/gif"
	"math/rand"
	"time"

	"github.com/andybons/gogif"
	"golang.org/x/image/draw"
)

func renderGif(solvedGame *game, moves []move) (bytes.Buffer, error) {
	cpy := solvedGame.clone()
	g := &cpy

	moveGif := gif.GIF{LoopCount: 0}

	var images []draw.Image
	var delays []int

	/*
		// first frame
		img, err := render(g)
		if err != nil {
			panic(err)
		}

		var test bytes.Buffer
		if err := png.Encode(&test, img); err != nil {
			log.Fatalf("Encoding: %v", err)
		}
		ioutil.WriteFile("dave.png", test.Bytes(), 0666)
	*/

	boardImg, err := renderGifBoard(g)
	if err != nil {
		return bytes.Buffer{}, nil
	}

	rand.Seed(time.Now().UnixNano())
	for _, m := range moves {

		// move robot
		startPos := g.robots[m.id].position
		g.move(g.robots[m.id], m.dir)
		endPos := g.robots[m.id].position

		// now draw robot at start + several discrete steps + end
		startRow := startPos / 16
		startCol := startPos % 16
		startX := int(startCol * 16)
		startY := int(startRow * 16)

		endRow := endPos / 16
		endCol := endPos % 16
		endX := int(endCol * 16)
		endY := int(endRow * 16)

		// start
		boardWithOtherRobots := copyImg(boardImg)
		drawRobots(g, m.id, boardWithOtherRobots)
		robotImg := pickRobot(*g.robots[m.id])

		cpy := copyImg(boardWithOtherRobots)
		draw.Draw(cpy, image.Rect(startX, startY, startX+16, startY+16), robotImg, image.Point{}, draw.Over)
		images = append(images, cpy)
		delays = append(delays, 20)

		xDiff := endX - startX
		yDiff := endY - startY

		// middle frames
		numSteps := 3
		for step := 0; step < numSteps; step++ {
			cpy := copyImg(boardWithOtherRobots)
			interpX := (xDiff / (numSteps + 2)) * (step + 1)
			interpY := (yDiff / (numSteps + 2)) * (step + 1)
			draw.Draw(cpy, image.Rect(startX+interpX, startY+interpY, startX+interpX+16, startY+interpY+16), robotImg, image.Point{}, draw.Over)

			images = append(images, cpy)
			delays = append(delays, 5)
		}

		// end
		cpy = copyImg(boardWithOtherRobots)
		draw.Draw(cpy, image.Rect(endX, endY, endX+16, endY+16), robotImg, image.Point{}, draw.Over)
		images = append(images, cpy)
		delays = append(delays, 20)
	}

	// the default stdlib gif palette indexer is extremely slow so use this one instead
	quantizer := gogif.MedianCutQuantizer{NumColor: 64}

	// convert to gif compatible images
	for idx, img := range images {
		dst := image.NewPaletted(image.Rect(0, 0, 16*16, 16*16), palette.WebSafe)
		quantizer.Quantize(dst, dst.Bounds(), img, image.Point{})

		moveGif.Image = append(moveGif.Image, dst)
		moveGif.Delay = append(moveGif.Delay, delays[idx])
	}

	var out bytes.Buffer
	gif.EncodeAll(&out, &moveGif)

	/*
		if err := ioutil.WriteFile("out.gif", out.Bytes(), 0666); err != nil {
			panic(err)
		}
	*/
	return out, nil
}

func copyImg(img draw.Image) draw.Image {
	dst := image.NewNRGBA(image.Rect(0, 0, 16*16, 16*16))
	draw.Draw(dst, img.Bounds(), img, image.Point{}, draw.Over)
	return dst
}

func drawRobots(g *game, ignore byte, dst draw.Image) {
	for _, r := range g.robots {
		if r.id == ignore {
			continue
		}
		robotImg := pickRobot(*r)
		row := r.position / 16
		col := r.position % 16

		x := int(col * 16)
		y := int(row * 16)
		draw.Draw(dst, image.Rect(x, y, x+16, y+16), robotImg, image.Point{}, draw.Over)
	}
}

// renders board + goal but no robots
func renderGifBoard(g *game) (draw.Image, error) {

	dst := image.NewNRGBA(image.Rect(0, 0, 16*16, 16*16))
	// one row at a time
	for row := 0; row < g.size; row += 1 {
		for col := 0; col < g.size; col += 1 {
			sq := g.board[row*g.size+col]
			tile := pickTile(sq)

			x := (col * 16)
			y := (row * 16)
			draw.Draw(dst, image.Rect(x, y, x+16, y+16), tile, image.Point{}, draw.Over)
		}
	}

	// draw goal
	goalImg := pickGoal(g.activeGoal)
	row := g.activeGoal.position / 16
	col := g.activeGoal.position % 16

	x := int(col * 16)
	y := int(row * 16)
	draw.Draw(dst, image.Rect(x, y, x+16, y+16), goalImg, image.Point{}, draw.Over)

	return dst, nil
}
