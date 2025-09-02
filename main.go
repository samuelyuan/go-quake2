package main

import (
	"log"
	"runtime"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/samuelyuan/go-quake2/engine"
)

func main() {
	log.Println("Starting Quake 2 BSP Loader...")

	// Lock OS thread for OpenGL operations
	runtime.LockOSThread()

	// Initialize GLFW
	if err := glfw.Init(); err != nil {
		log.Fatalf("Failed to initialize GLFW: %v", err)
	}
	defer glfw.Terminate()

	// Create and run the game
	game, err := engine.NewGame()
	if err != nil {
		log.Fatalf("Failed to create game: %v", err)
	}

	// Run the main game loop
	if err := game.Run(); err != nil {
		log.Fatalf("Game error: %v", err)
	}

	log.Println("Game terminated successfully")
}
