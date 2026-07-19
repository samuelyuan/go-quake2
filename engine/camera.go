package engine

import (
	"math"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/samuelyuan/go-quake2/client"
)

// Camera represents a 3D camera with position and orientation
type Camera struct {
	xAngle         float32
	zAngle         float32
	cameraPosition mgl32.Vec3
	windowHandler  *client.WindowHandler
}

// NewCamera creates a new camera instance
func NewCamera(windowHandler *client.WindowHandler) *Camera {
	return &Camera{
		xAngle:         0.0,
		zAngle:         float32(math.Pi), // Start facing forward
		cameraPosition: mgl32.Vec3{-50, 256, -50},
		windowHandler:  windowHandler,
	}
}

// GetViewMatrix returns the camera's view matrix
func (c *Camera) GetViewMatrix() mgl32.Mat4 {
	matrix := mgl32.Ident4()
	matrix = matrix.Mul4(mgl32.HomogRotate3DX(c.xAngle - mgl32.DegToRad(90)))
	matrix = matrix.Mul4(mgl32.HomogRotate3DZ(c.zAngle))
	matrix = matrix.Mul4(mgl32.Translate3D(c.cameraPosition.X(), c.cameraPosition.Y(), c.cameraPosition.Z()))
	return matrix
}

// GetPerspectiveMatrix returns the camera's perspective projection matrix
func (c *Camera) GetPerspectiveMatrix() mgl32.Mat4 {
	ratio := float64(c.windowHandler.GetWidth()) / float64(c.windowHandler.GetHeight())
	return mgl32.Perspective(
		float32(CameraFOV),
		float32(ratio),
		float32(NearPlane),
		float32(FarPlane),
	)
}

// UpdateViewMatrix updates the camera position and orientation based on input
func (c *Camera) UpdateViewMatrix() {
	// Handle keyboard movement
	c.handleKeyboardMovement()

	// Handle mouse look
	c.handleMouseLook()
}

// handleKeyboardMovement processes WASD movement input
func (c *Camera) handleKeyboardMovement() {
	speed := float32(CameraSpeed * c.windowHandler.GetTimeSinceLastFrame())

	// Calculate movement direction
	dir := mgl32.Vec3{0, 0, 0}
	if c.windowHandler.InputHandler.IsActive(client.PLAYER_FORWARD) {
		dir[2] += speed
	} else if c.windowHandler.InputHandler.IsActive(client.PLAYER_BACKWARD) {
		dir[2] -= speed
	}

	if c.windowHandler.InputHandler.IsActive(client.PLAYER_LEFT) {
		dir[0] += speed
	} else if c.windowHandler.InputHandler.IsActive(client.PLAYER_RIGHT) {
		dir[0] -= speed
	}

	// Apply movement in camera space
	cameraMatrix := mgl32.Ident4()
	cameraMatrix = cameraMatrix.Mul4(mgl32.HomogRotate3DX(c.xAngle - mgl32.DegToRad(90)))
	cameraMatrix = cameraMatrix.Mul4(mgl32.HomogRotate3DZ(c.zAngle))
	cameraMatrix = cameraMatrix.Inv()

	movementDelta := cameraMatrix.Mul4x1(mgl32.Vec4{dir[0], dir[1], dir[2], 0.0})
	c.cameraPosition = c.cameraPosition.Add(mgl32.Vec3{movementDelta.X(), movementDelta.Y(), movementDelta.Z()})
}

// handleMouseLook processes mouse input for camera rotation
func (c *Camera) handleMouseLook() {
	offset := c.windowHandler.InputHandler.GetCursorChange()
	xOffset := float32(offset[0] * MouseSensitivity)
	yOffset := float32(offset[1] * MouseSensitivity)

	// Update horizontal rotation (Yaw)
	c.zAngle += xOffset * 0.025
	c.zAngle = c.normalizeAngle(c.zAngle)

	// Update vertical rotation (Pitch)
	c.xAngle += yOffset * 0.025
	c.xAngle = c.clampAngle(c.xAngle, -math.Pi*0.5, math.Pi*0.5)
}

// normalizeAngle keeps an angle within [0, 2π)
func (c *Camera) normalizeAngle(angle float32) float32 {
	for angle < 0 {
		angle += math.Pi * 2
	}
	for angle >= math.Pi*2 {
		angle -= math.Pi * 2
	}
	return angle
}

// clampAngle constrains an angle between min and max values
func (c *Camera) clampAngle(angle, min, max float32) float32 {
	if angle < min {
		return min
	}
	if angle > max {
		return max
	}
	return angle
}

// GetCameraPosition returns the camera's current position
func (c *Camera) GetCameraPosition() [3]float32 {
	position := c.cameraPosition
	return [3]float32{-position.X(), -position.Y(), -position.Z()}
}
