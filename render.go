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
			f, err := os.Open(filepath.Join("ricochet-images", "vanilla3.png"))
			if err != nil {
				log.Fatalf("opnening tile: %v", err)
			}
			tile, err := png.Decode(f)
			if err != nil {
				log.Fatalf("decoding tile: %v", err)
			}

			x := (col * 16)
			y := (row * 16)
			fmt.Println("x:", x, " y:", y)
			draw.BiLinear.Scale(dst, image.Rect(x, y, x+16, y+16), tile, tile.Bounds(), draw.Over, nil)
		}
	}

	out, err := os.Create("out.png")
	if err != nil {
		log.Fatalf("creating output: %v", err)
	}
	if err := png.Encode(out, dst); err != nil {
		log.Fatalf("Encoding: %v", err)
	}

	return dst, nil
}
