package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"

	"github.com/samuelyuan/go-quake2/render"
)

func main() {
	// stored in different folder
	// format: ./testWal inputFilename outputFilename
	fullFilename := "../data/textures/" + os.Args[1]

	texFile, _ := os.Open(fullFilename)
	defer texFile.Close()

	if texFile == nil {
		log.Fatal("Texture file doesn't exist")
		return
	}

	imageData, walData, err := render.LoadQ2WAL(texFile)
	if err != nil {
		log.Fatal("Error loading texture in main:", err)
		return
	}

	fmt.Println("Successfully loaded WAL file at " + fullFilename)

	// Create new image
	var imgData = image.NewRGBA(image.Rect(0, 0, int(walData.Width), int(walData.Height)))

	// Save each pixel to the image
	byteCount := walData.Width * walData.Height * 3
	for i := 0; i < int(byteCount); i += 3 {
		pixelId := i / 3
		y := pixelId / int(walData.Width)
		x := pixelId % int(walData.Width)

		// add new color
		// each integer represents r, g, b respectively
		r := imageData[i]
		g := imageData[i+1]
		b := imageData[i+2]
		imgData.Set(x, y, color.RGBA{r, g, b, 255})
	}

	outputFilename := os.Args[2]
	imageOutputFile, err := os.Create(outputFilename)
	if err != nil {
		panic(err)
	}
	defer imageOutputFile.Close()
	png.Encode(imageOutputFile, imgData)

	fmt.Println("Written image data to " + outputFilename)
}
