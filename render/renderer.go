package render

import (
	"fmt"

	"github.com/go-gl/mathgl/mgl32"
)

type Renderer struct {
	glState    *GLStateManager
	buffers    *BufferManager
	textureMgr *TextureManager
	shader     *Shader
	uniforms   *UniformManager
	skybox     *Skybox
}

func NewRenderer() *Renderer {
	return &Renderer{
		glState:    NewGLStateManager(),
		buffers:    NewBufferManager(),
		textureMgr: NewTextureManager(),
		shader:     nil, // Will be set in Init()
		uniforms:   nil, // Will be set in Init()
	}
}

func (r *Renderer) Init() error {
	// Initialize OpenGL state
	if err := r.glState.Init(); err != nil {
		return fmt.Errorf("failed to initialize OpenGL state: %w", err)
	}

	// Create shader
	shader, err := NewShader("shaders/goquake2.vert", "shaders/goquake2.frag")
	if err != nil {
		return fmt.Errorf("failed to create shader: %w", err)
	}
	r.shader = shader

	// Create uniform manager
	r.uniforms = NewUniformManager(r.shader.ProgramShader)

	// Initialize buffers
	r.buffers.Init()

	return nil
}

func (r *Renderer) PrepareFrame(viewMatrix mgl32.Mat4, projectionMatrix mgl32.Mat4) {
	// Clear buffers
	r.glState.ClearBuffers()

	// Use shader program
	r.uniforms.UseProgram()

	// Set camera matrices
	r.uniforms.SetViewMatrix(viewMatrix)
	r.uniforms.SetProjectionMatrix(projectionMatrix)
}

// GetTextureManager returns the texture manager
func (r *Renderer) GetTextureManager() *TextureManager {
	return r.textureMgr
}

// SetSkybox sets the skybox for this renderer
func (r *Renderer) SetSkybox(skybox *Skybox) {
	r.skybox = skybox
}

// DrawSkybox draws the skybox if one is set
func (r *Renderer) DrawSkybox() {
	if r.skybox != nil {
		r.skybox.Draw()
	}
}

// Cleanup frees all OpenGL resources
func (r *Renderer) Cleanup() {
	if r.buffers != nil {
		r.buffers.Cleanup()
	}
	if r.textureMgr != nil {
		r.textureMgr.Cleanup()
	}
	if r.shader != nil {
		r.shader.Cleanup()
	}
}
