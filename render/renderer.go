package render

import (
	"fmt"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

type Renderer struct {
	Vao    uint32
	Vbo    uint32
	Shader *Shader
}

func NewRenderer() *Renderer {
	return &Renderer{}
}

func (r *Renderer) Init() {
	if err := gl.Init(); err != nil {
		panic(err)
	}

	version := gl.GoStr(gl.GetString(gl.VERSION))
	fmt.Println("OpenGL version", version)

	r.Shader = NewShader("render/goquake2.vert", "render/goquake2.frag")

	gl.ClearColor(0.0, 0.0, 0.0, 1.0)
	gl.Enable(gl.DEPTH_TEST)

	// Set appropriate blending mode
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	gl.Enable(gl.CULL_FACE)
	gl.CullFace(gl.FRONT)

	// Create buffers/arrays
	gl.GenVertexArrays(1, &r.Vao)
	gl.GenBuffers(1, &r.Vbo)
}

func (r *Renderer) PrepareFrame(viewMatrix mgl32.Mat4, projectionMatrix mgl32.Mat4) {
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

	programShader := r.Shader.ProgramShader

	gl.UseProgram(programShader)

	// Pass the camera matrices to the shader
	viewLoc := gl.GetUniformLocation(programShader, gl.Str("view\x00"))
	gl.UniformMatrix4fv(viewLoc, 1, false, &viewMatrix[0])

	projectionLoc := gl.GetUniformLocation(programShader, gl.Str("projection\x00"))
	gl.UniformMatrix4fv(projectionLoc, 1, false, &projectionMatrix[0])
}
