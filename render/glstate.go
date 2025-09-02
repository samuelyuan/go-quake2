package render

import (
	"fmt"

	"github.com/go-gl/gl/v4.1-core/gl"
)

// GLStateManager handles OpenGL state initialization and configuration
type GLStateManager struct {
	initialized bool
}

// NewGLStateManager creates a new OpenGL state manager
func NewGLStateManager() *GLStateManager {
	return &GLStateManager{}
}

// Init initializes OpenGL and sets up global state
func (gsm *GLStateManager) Init() error {
	if gsm.initialized {
		return nil
	}

	if err := gl.Init(); err != nil {
		return fmt.Errorf("failed to initialize OpenGL: %v", err)
	}

	version := gl.GoStr(gl.GetString(gl.VERSION))
	fmt.Println("OpenGL version", version)

	gsm.setupDefaultState()
	gsm.initialized = true
	return nil
}

// setupDefaultState configures the default OpenGL rendering state
func (gsm *GLStateManager) setupDefaultState() {
	// Set clear color (black background)
	gl.ClearColor(0.0, 0.0, 0.0, 1.0)

	// Enable depth testing
	gl.Enable(gl.DEPTH_TEST)

	// Enable blending for transparency
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	// Enable face culling for performance
	gl.Enable(gl.CULL_FACE)
	gl.CullFace(gl.FRONT)
}

// ClearBuffers clears the color and depth buffers
func (gsm *GLStateManager) ClearBuffers() {
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
}

// IsInitialized returns whether OpenGL has been initialized
func (gsm *GLStateManager) IsInitialized() bool {
	return gsm.initialized
}
