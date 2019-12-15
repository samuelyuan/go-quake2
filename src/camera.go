package main

import (
	"github.com/go-gl/mathgl/mgl32"
	"math"
)

const (
	MouseSensitivity = 0.7
)

type Camera struct {
	xAngle         float32
	zAngle         float32
	cameraPosition mgl32.Vec3
	windowHandler  *WindowHandler
}

func NewCamera(windowHandler *WindowHandler) *Camera {
	return &Camera{
		xAngle:         float32(0),
		zAngle:         float32(3),
		cameraPosition: mgl32.Vec3{-50, 256, -50},
		windowHandler:  windowHandler,
	}
}

func (c *Camera) GetViewMatrix() mgl32.Mat4 {
	matrix := mgl32.Ident4()
	matrix = matrix.Mul4(mgl32.HomogRotate3DX(c.xAngle - mgl32.DegToRad(90)))
	matrix = matrix.Mul4(mgl32.HomogRotate3DZ(c.zAngle))
	matrix = matrix.Mul4(mgl32.Translate3D(c.cameraPosition.X(), c.cameraPosition.Y(), c.cameraPosition.Z()))
	return matrix
}

func (c *Camera) GetPerspectiveMatrix() mgl32.Mat4 {
	ratio := float64(windowWidth) / float64(windowHeight)
	return mgl32.Perspective(45.0, float32(ratio), 0.1, 4096.0)
}

func (c *Camera) UpdateViewMatrix() {
	// Move the camera around using WASD keys
	speed := float32(200 * c.windowHandler.getTimeSinceLastFrame())
	dir := []float32{0, 0, 0}
	if c.windowHandler.inputHandler.isActive(PLAYER_FORWARD) {
		dir[2] += speed
	} else if c.windowHandler.inputHandler.isActive(PLAYER_BACKWARD) {
		dir[2] -= speed
	} else if c.windowHandler.inputHandler.isActive(PLAYER_LEFT) {
		dir[0] += speed
	} else if c.windowHandler.inputHandler.isActive(PLAYER_RIGHT) {
		dir[0] -= speed
	}

	cameraMatrix := mgl32.Ident4()
	cameraMatrix = cameraMatrix.Mul4(mgl32.HomogRotate3DX(c.xAngle - mgl32.DegToRad(90)))
	cameraMatrix = cameraMatrix.Mul4(mgl32.HomogRotate3DZ(c.zAngle))
	cameraMatrix = cameraMatrix.Inv()
	movementDelta := cameraMatrix.Mul4x1(mgl32.Vec4{dir[0], dir[1], dir[2], 0.0})

	c.cameraPosition = c.cameraPosition.Add(mgl32.Vec3{movementDelta.X(), movementDelta.Y(), movementDelta.Z()})

	offset := c.windowHandler.inputHandler.getCursorChange()
	xOffset := float32(offset[0] * MouseSensitivity)
	yOffset := float32(offset[1] * MouseSensitivity)

	c.zAngle += xOffset * 0.025
	for c.zAngle < 0 {
		c.zAngle += math.Pi * 2
	}
	for c.zAngle >= math.Pi*2 {
		c.zAngle -= math.Pi * 2
	}

	c.xAngle += yOffset * 0.025
	for c.xAngle < -math.Pi*0.5 {
		c.xAngle = -math.Pi * 0.5
	}
	for c.xAngle > math.Pi*0.5 {
		c.xAngle = math.Pi * 0.5
	}
}

func (c *Camera) GetCameraPosition() [3]float32 {
	position := c.cameraPosition
	return [3]float32{-position.X(), -position.Y(), -position.Z()}
}
