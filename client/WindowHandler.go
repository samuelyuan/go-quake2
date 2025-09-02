package client

import (
	"fmt"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

type WindowHandler struct {
	glfwWindow   *glfw.Window
	InputHandler *InputHandler

	firstFrame    bool
	deltaTime     float64
	lastFrameTime float64
}

// NewWindowHandler creates a new window handler with OpenGL context
func NewWindowHandler(width, height int, title string) *WindowHandler {
	// Set OpenGL context hints
	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.Resizable, glfw.True)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

	// Create the window
	glfwWindow, err := glfw.CreateWindow(width, height, title, nil, nil)
	if err != nil {
		panic(fmt.Errorf("failed to create OpenGL window: %v", err))
	}
	glfwWindow.MakeContextCurrent()

	// Set up callbacks
	glfwWindow.SetSizeCallback(resizeCallback)
	glfwWindow.GetSize()

	// Create input handler
	inputHandler := NewInputHandler()
	glfwWindow.SetKeyCallback(inputHandler.keyCallback)
	glfwWindow.SetCursorPosCallback(inputHandler.mouseCallback)

	return &WindowHandler{
		glfwWindow:   glfwWindow,
		InputHandler: inputHandler,
		firstFrame:   true,
	}
}

// resizeCallback handles window resize events
func resizeCallback(w *glfw.Window, width int, height int) {
	gl.Viewport(0, 0, int32(width), int32(height))
}

// StartFrame begins a new frame, handling input and timing
func (windowHandler *WindowHandler) StartFrame() {
	windowHandler.glfwWindow.SwapBuffers()

	// Process window events
	glfw.PollEvents()

	// Check for quit command
	if windowHandler.InputHandler.IsActive(PROGRAM_QUIT) {
		windowHandler.glfwWindow.SetShouldClose(true)
	}

	// Calculate frame timing
	currentFrameTime := glfw.GetTime()
	if windowHandler.firstFrame {
		windowHandler.lastFrameTime = currentFrameTime
		windowHandler.firstFrame = false
	}

	windowHandler.deltaTime = currentFrameTime - windowHandler.lastFrameTime
	windowHandler.lastFrameTime = currentFrameTime

	// Update cursor state
	windowHandler.InputHandler.updateCursor()
}

func (windowHandler *WindowHandler) ShouldClose() bool {
	return windowHandler.glfwWindow.ShouldClose()
}

func (windowHandler *WindowHandler) GetTimeSinceLastFrame() float64 {
	return windowHandler.deltaTime
}

// GetWidth returns the current window width
func (windowHandler *WindowHandler) GetWidth() int {
	width, _ := windowHandler.glfwWindow.GetSize()
	return width
}

// GetHeight returns the current window height
func (windowHandler *WindowHandler) GetHeight() int {
	_, height := windowHandler.glfwWindow.GetSize()
	return height
}
