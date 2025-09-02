package client

import (
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl64"
)

// Action represents a player or program action
type Action int

const (
	PLAYER_FORWARD  Action = iota // W key - move forward
	PLAYER_BACKWARD Action = iota // S key - move backward
	PLAYER_LEFT     Action = iota // A key - move left
	PLAYER_RIGHT    Action = iota // D key - move right
	PROGRAM_QUIT    Action = iota // Escape key - quit program
)

type InputHandler struct {
	actionToKeyMap map[Action]glfw.Key
	keysPressed    [glfw.KeyLast]bool

	firstCursorAction    bool
	cursor               mgl64.Vec2
	cursorChange         mgl64.Vec2
	cursorLast           mgl64.Vec2
	bufferedCursorChange mgl64.Vec2
}

// NewInputHandler creates a new input handler with default key bindings
func NewInputHandler() *InputHandler {
	actionToKeyMap := map[Action]glfw.Key{
		PLAYER_FORWARD:  glfw.KeyW,
		PLAYER_BACKWARD: glfw.KeyS,
		PLAYER_LEFT:     glfw.KeyA,
		PLAYER_RIGHT:    glfw.KeyD,
		PROGRAM_QUIT:    glfw.KeyEscape,
	}

	return &InputHandler{
		actionToKeyMap:    actionToKeyMap,
		firstCursorAction: false,
	}
}

// IsActive checks if a specific action is currently active
func (handler *InputHandler) IsActive(a Action) bool {
	return handler.keysPressed[handler.actionToKeyMap[a]]
}

// keyCallback handles keyboard input events
func (handler *InputHandler) keyCallback(window *glfw.Window, key glfw.Key, scancode int,
	action glfw.Action, mods glfw.ModifierKey) {

	switch action {
	case glfw.Press:
		handler.keysPressed[key] = true
	case glfw.Release:
		handler.keysPressed[key] = false
	}
}

// GetCursorChange returns the cursor movement delta since last frame
func (handler *InputHandler) GetCursorChange() mgl64.Vec2 {
	return handler.cursorChange
}

// updateCursor updates cursor state and resets movement delta
func (handler *InputHandler) updateCursor() {
	handler.cursorChange[0] = handler.bufferedCursorChange[0]
	handler.cursorChange[1] = handler.bufferedCursorChange[1]
	handler.cursor[0] = handler.cursorLast[0]
	handler.cursor[1] = handler.cursorLast[1]

	handler.bufferedCursorChange[0] = 0
	handler.bufferedCursorChange[1] = 0
}

// mouseCallback handles mouse movement events
func (handler *InputHandler) mouseCallback(window *glfw.Window, xPos float64, yPos float64) {
	if handler.firstCursorAction {
		handler.cursorLast[0] = xPos
		handler.cursorLast[1] = yPos
		handler.firstCursorAction = false
	}

	handler.bufferedCursorChange[0] += xPos - handler.cursorLast[0]
	handler.bufferedCursorChange[1] += handler.cursorLast[1] - yPos

	handler.cursorLast[0] = xPos
	handler.cursorLast[1] = yPos
}
