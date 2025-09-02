package render

import (
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

// UniformManager handles shader uniform operations
type UniformManager struct {
	programID uint32
}

// NewUniformManager creates a new uniform manager for a shader program
func NewUniformManager(programID uint32) *UniformManager {
	return &UniformManager{
		programID: programID,
	}
}

// SetViewMatrix sets the view matrix uniform
func (um *UniformManager) SetViewMatrix(viewMatrix mgl32.Mat4) {
	viewLoc := gl.GetUniformLocation(um.programID, gl.Str("view\x00"))
	gl.UniformMatrix4fv(viewLoc, 1, false, &viewMatrix[0])
}

// SetProjectionMatrix sets the projection matrix uniform
func (um *UniformManager) SetProjectionMatrix(projectionMatrix mgl32.Mat4) {
	projectionLoc := gl.GetUniformLocation(um.programID, gl.Str("projection\x00"))
	gl.UniformMatrix4fv(projectionLoc, 1, false, &projectionMatrix[0])
}

// SetDiffuseTexture sets the diffuse texture uniform
func (um *UniformManager) SetDiffuseTexture(textureUnit int32) {
	diffuseUniform := gl.GetUniformLocation(um.programID, gl.Str("diffuse\x00"))
	gl.Uniform1i(diffuseUniform, textureUnit)
}

// SetLightmapTexture sets the lightmap texture uniform
func (um *UniformManager) SetLightmapTexture(textureUnit int32) {
	lightmapUniform := gl.GetUniformLocation(um.programID, gl.Str("lightmap\x00"))
	gl.Uniform1i(lightmapUniform, textureUnit)
}

// UseProgram activates the shader program
func (um *UniformManager) UseProgram() {
	gl.UseProgram(um.programID)
}
