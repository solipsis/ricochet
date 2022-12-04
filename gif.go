package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color/palette"
	"image/gif"
	"image/png"
	"io/ioutil"
	"log"
	"math/rand"
	"time"

	"golang.org/x/image/draw"
)

func renderGif(solvedGame *game, moves []move) {
	cpy := solvedGame.clone()
	g := &cpy

	moveGif := gif.GIF{LoopCount: 0}

	// first frame
	img, err := render(g)
	if err != nil {
		panic(err)
	}
	/*
		pm := image.NewPaletted(img.Bounds(), palette.Plan9)
		draw.Draw(pm, img.Bounds(), img, image.Point{}, draw.Over)
		moveGif.Image = append(moveGif.Image, pm)
		moveGif.Delay = append(moveGif.Delay, 50)
	*/

	var test bytes.Buffer
	if err := png.Encode(&test, img); err != nil {
		log.Fatalf("Encoding: %v", err)
	}
	ioutil.WriteFile("dave.png", test.Bytes(), 0666)

	// get keyframe == state sans robot we are moving
	// draw robot and start + x discrete steps + end

	rand.Seed(time.Now().UnixNano())
	for _, m := range moves {
		fmt.Println("M:", m.String())

		startPos := g.robots[m.id].position

		// toggle off the robot we are moving and then render without it
		//g.board[startPos] = g.board[startPos] &^ square(ROBOT)

		keyframe, err := renderGifFrame(g, m.id)
		if err != nil {
			panic(err)
		}

		// toggle robot back on and move
		//g.board[startPos] = g.board[startPos] | square(ROBOT)
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
		cpy := copyImg(keyframe)
		robotImg := pickRobot(*g.robots[m.id])
		draw.Draw(cpy, image.Rect(startX, startY, startX+16, startY+16), robotImg, image.Point{}, draw.Over)
		moveGif.Image = append(moveGif.Image, cpy)
		moveGif.Delay = append(moveGif.Delay, 25)

		xDiff := endX - startX
		yDiff := endY - startY

		// middle frames
		numSteps := 5
		for step := 0; step < numSteps; step++ {
			cpy := copyImg(keyframe)
			interpX := (xDiff / (numSteps + 2)) * (step + 1)
			interpY := (yDiff / (numSteps + 2)) * (step + 1)
			draw.Draw(cpy, image.Rect(startX+interpX, startY+interpY, startX+interpX+16, startY+interpY+16), robotImg, image.Point{}, draw.Over)
			moveGif.Image = append(moveGif.Image, cpy)
			moveGif.Delay = append(moveGif.Delay, 5)
		}

		// end
		cpy = copyImg(keyframe)
		draw.Draw(cpy, image.Rect(endX, endY, endX+16, endY+16), robotImg, image.Point{}, draw.Over)
		moveGif.Image = append(moveGif.Image, cpy)
		moveGif.Delay = append(moveGif.Delay, 25)

	}

	var out bytes.Buffer
	gif.EncodeAll(&out, &moveGif)

	if err := ioutil.WriteFile("out.gif", out.Bytes(), 0666); err != nil {
		panic(err)
	}

}

func copyImg(img draw.Image) *image.Paletted {

	dst := image.NewPaletted(img.Bounds(), palette.Plan9)
	draw.Draw(dst, img.Bounds(), img, image.Point{}, draw.Over)
	return dst
}

func renderGifFrame(g *game, ignore byte) (draw.Image, error) {

	dst := image.NewNRGBA(image.Rect(0, 0, 16*16, 16*16))
	// one row at a time
	for row := 0; row < g.size; row += 1 {
		for col := 0; col < g.size; col += 1 {
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

	// draw goal
	goalImg := pickGoal(g.activeGoal)
	row := g.activeGoal.position / 16
	col := g.activeGoal.position % 16

	x := int(col * 16)
	y := int(row * 16)
	draw.BiLinear.Scale(dst, image.Rect(x, y, x+16, y+16), goalImg, goalImg.Bounds(), draw.Over, nil)

	// draw robots
	for _, r := range g.robots {

		if ignore == r.id {
			continue
		}

		robotImg := pickRobot(*r)
		row := r.position / 16
		col := r.position % 16

		x := int(col * 16)
		y := int(row * 16)
		draw.BiLinear.Scale(dst, image.Rect(x, y, x+16, y+16), robotImg, robotImg.Bounds(), draw.Over, nil)
	}

	return dst, nil
}
