package render

import (
	"github.com/go-gl/gl/v4.1-core/gl"
)

// Skybox represents a simple background color clear
type Skybox struct {
	// No fields needed for simple background clear
}

// NewSkybox creates a new skybox
func NewSkybox(textureID uint32) *Skybox {
	// textureID parameter kept for future compatibility
	return &Skybox{}
}

// Draw renders the skybox (just clears to dark burnt orange/red)
func (s *Skybox) Draw(viewMatrix, projectionMatrix interface{}) {
	// Parameters kept for future compatibility but not used
	gl.ClearColor(0.3, 0.1, 0.05, 1.0) // Dark burnt orange/red - authentic Quake 2
	gl.Clear(gl.COLOR_BUFFER_BIT)
}

// Cleanup frees OpenGL resources (none to clean up)
func (s *Skybox) Cleanup() {
	// Nothing to clean up for simple background clear
}
