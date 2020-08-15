package render

import (
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/samuelyuan/go-quake2/q2file"
)

type MapTexture struct {
	Id         uint32
	Width      uint32
	Height     uint32
	VertOffset int32
	VertCount  int32
}

func NewMapTexture(id uint32, width uint32, height uint32) MapTexture {
	texture := MapTexture{}
	texture.Id = id
	texture.Width = width
	texture.Height = height
	return texture
}

// Initialize texture in OpenGL using image data
func BuildWALTexture(imageData []uint8, walData q2file.WalHeader) uint32 {
	var texId uint32
	gl.GenTextures(1, &texId)
	gl.BindTexture(gl.TEXTURE_2D, texId)

	// Give the image to OpenGL
	gl.TexImage2D(uint32(gl.TEXTURE_2D), 0, int32(gl.RGB), int32(walData.Width), int32(walData.Height),
		0, uint32(gl.RGB), uint32(gl.UNSIGNED_BYTE), gl.Ptr(imageData))

	// Set texture wrapping/filtering options
	gl.TexParameteri(uint32(gl.TEXTURE_2D), gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(uint32(gl.TEXTURE_2D), gl.TEXTURE_MIN_FILTER, gl.LINEAR)

	return texId
}
