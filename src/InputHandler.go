package main

import (
	"github.com/go-gl/glfw/v3.2/glfw"
)

type Action int

const (
	PLAYER_FORWARD  Action = iota
	PLAYER_BACKWARD Action = iota
	PLAYER_LEFT     Action = iota
	PLAYER_RIGHT    Action = iota
	PROGRAM_QUIT    Action = iota
)

type InputHandler struct {
	actionToKeyMap map[Action]glfw.Key
	keysPressed [glfw.KeyLast]bool
}

func NewInputHandler() *InputHandler {
	actionToKeyMap := map[Action]glfw.Key{
		PLAYER_FORWARD:  glfw.KeyW,
		PLAYER_BACKWARD: glfw.KeyS,
		PLAYER_LEFT:     glfw.KeyA,
		PLAYER_RIGHT:    glfw.KeyD,
		PROGRAM_QUIT:    glfw.KeyEscape,
	}

	return &InputHandler{
		actionToKeyMap: actionToKeyMap,
	}
}

func (handler *InputHandler) isActive(a Action) bool {
	return handler.keysPressed[handler.actionToKeyMap[a]]
}

func (handler *InputHandler) keyCallback(window *glfw.Window, key glfw.Key, scancode int,
	action glfw.Action, mods glfw.ModifierKey) {

	switch action {
	case glfw.Press:
		handler.keysPressed[key] = true
	case glfw.Release:
		handler.keysPressed[key] = false
	}
}
