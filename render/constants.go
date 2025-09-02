package render

import (
	"github.com/go-gl/gl/v4.1-core/gl"
)

// Lightmap constants
const (
	LIGHTMAP_SIZE = int32(512)
)

// OpenGL texture constants
const (
	DEFAULT_TEXTURE_MAG_FILTER  = gl.LINEAR
	DEFAULT_TEXTURE_MIN_FILTER  = gl.LINEAR
	LIGHTMAP_TEXTURE_MIN_FILTER = gl.LINEAR_MIPMAP_LINEAR
)

// Vertex attribute locations
const (
	POSITION_ATTRIB_LOCATION = 0
	TEXCOORD_ATTRIB_LOCATION = 1
	LIGHTMAP_ATTRIB_LOCATION = 2
)

// Note: FLOAT_SIZE and TexturedVertexSize are defined in other files
