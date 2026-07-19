package render

import (
	"github.com/go-gl/gl/v4.1-core/gl"
)

// BufferManager handles all OpenGL buffer operations
type BufferManager struct {
	vao uint32
	vbo uint32
}

// NewBufferManager creates a new buffer manager
func NewBufferManager() *BufferManager {
	return &BufferManager{}
}

// Init creates and initializes the VAO and VBO
func (bm *BufferManager) Init() {
	gl.GenVertexArrays(1, &bm.vao)
	gl.GenBuffers(1, &bm.vbo)
}

// BindVertexArray binds the VAO for rendering
func (bm *BufferManager) BindVertexArray() {
	gl.BindVertexArray(bm.vao)
}

// BindVertexBuffer binds the VBO for data operations
func (bm *BufferManager) BindVertexBuffer() {
	gl.BindBuffer(gl.ARRAY_BUFFER, bm.vbo)
}

// UploadVertexData uploads vertex data to the VBO
func (bm *BufferManager) UploadVertexData(data []float32) {
	gl.BufferData(gl.ARRAY_BUFFER, len(data)*FLOAT_SIZE, gl.Ptr(data), gl.STATIC_DRAW)
}

// SetupVertexAttributes configures all vertex attributes for textured vertices
func (bm *BufferManager) SetupVertexAttributes() {
	// 3 floats for vertex, 2 floats for texture UV, 2 floats for lightmap UV
	stride := int32(TexturedVertexSize * FLOAT_SIZE)

	// Position attribute (location 0)
	gl.VertexAttribPointer(POSITION_ATTRIB_LOCATION, 3, gl.FLOAT, false, stride, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(POSITION_ATTRIB_LOCATION)

	// Texture UV attribute (location 1)
	gl.VertexAttribPointer(TEXCOORD_ATTRIB_LOCATION, 2, gl.FLOAT, false, stride, gl.PtrOffset(3*FLOAT_SIZE))
	gl.EnableVertexAttribArray(TEXCOORD_ATTRIB_LOCATION)

	// Lightmap UV attribute (location 2)
	gl.VertexAttribPointer(LIGHTMAP_ATTRIB_LOCATION, 2, gl.FLOAT, false, stride, gl.PtrOffset(5*FLOAT_SIZE))
	gl.EnableVertexAttribArray(LIGHTMAP_ATTRIB_LOCATION)
}

// Cleanup frees OpenGL resources
func (bm *BufferManager) Cleanup() {
	gl.DeleteVertexArrays(1, &bm.vao)
	gl.DeleteBuffers(1, &bm.vbo)
}
