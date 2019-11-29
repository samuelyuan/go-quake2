package main

import (
	"fmt"
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
)

type WindowHandler struct {
	glfwWindow   *glfw.Window
	inputHandler *InputHandler
}

func NewWindowHandler(width, height int, title string) *WindowHandler {
	if err := glfw.Init(); err != nil {
		panic(fmt.Errorf("Could not initialize glfw: %v", err))
	}

	// Initialize and create window
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.Resizable, glfw.True)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

	glfwWindow, err := glfw.CreateWindow(width, height, title, nil, nil)
	if err != nil {
		panic(fmt.Errorf("Could not create OpenGL renderer: %v", err))
	}
	glfwWindow.MakeContextCurrent()

	// Check for resize
	glfwWindow.SetSizeCallback(resizeCallback)
	glfwWindow.GetSize()

	inputHandler := NewInputHandler()

	// Keyboard callback
	glfwWindow.SetKeyCallback(inputHandler.keyCallback)
	// Mouse callback
	glfwWindow.SetCursorPosCallback(inputHandler.mouseCallback)

	return &WindowHandler{
		glfwWindow:   glfwWindow,
		inputHandler: inputHandler,
	}
}

// Resize the screen
func resizeCallback(w *glfw.Window, width int, height int) {
	gl.Viewport(0, 0, int32(width), int32(height))
}
