package render

import (
	"github.com/go-gl/gl/v4.1-core/gl"
)

const (
	FLOAT_SIZE = 4
)

func DrawMap(renderer *Renderer, renderMap RenderMap) {
	// Use buffer manager for all buffer operations
	renderer.buffers.BindVertexArray()
	renderer.buffers.BindVertexBuffer()
	renderer.buffers.UploadVertexData(renderMap.VertexBuffer)
	renderer.buffers.SetupVertexAttributes()

	// Set texture uniforms using the uniform manager
	renderer.uniforms.SetDiffuseTexture(0)

	// Bind the lightmap texture shared by all the faces
	gl.ActiveTexture(gl.TEXTURE1)
	gl.BindTexture(gl.TEXTURE_2D, (*renderMap.MapLightmap).Texture)
	renderer.uniforms.SetLightmapTexture(1)

	// Since faces are sorted by texture, we loop through all textures in the map
	mapTextures := renderMap.MapTextures
	for i := 0; i < len(mapTextures); i++ {
		texture := mapTextures[i]

		if texture.VertCount == 0 {
			continue
		}

		// Bind the texture
		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D, texture.Id)

		// Draw all faces for this texture
		gl.DrawArrays(gl.TRIANGLES, texture.VertOffset, texture.VertCount)
	}

	return
}
