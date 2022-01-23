package render

import (
	"sort"
)

const (
	TexturedVertexSize = 7
)

// Convert map data to a float array for rendering
type PolygonBuffer struct {
	Buffer      []float32 // Contains vertices, texture UV, lightmap UV
	MapTextures []MapTexture
}

// Rearrange data by texture
func NewPolygonBuffer(surfacesByTexture map[int][]Surface, mapTextures []MapTexture) *PolygonBuffer {
	// only get the textures that were used in the map
	var texKeys []int
	for k, _ := range surfacesByTexture {
		texKeys = append(texKeys, k)
	}
	sort.Ints(texKeys)

	// allocate a buffer
	bufferSize := 0
	for _, textureId := range texKeys {
		for _, surface := range surfacesByTexture[textureId] {
			// Each element has 7 floats
			bufferSize += int(len(surface.TexturedVertices)) * TexturedVertexSize
		}
	}

	polygonBuffer := &PolygonBuffer{}
	polygonBuffer.MapTextures = make([]MapTexture, len(mapTextures))
	// Copy
	for index, mapTexture := range mapTextures {
		polygonBuffer.MapTextures[index] = mapTexture
		polygonBuffer.MapTextures[index].VertOffset = 0
		polygonBuffer.MapTextures[index].VertCount = int32(0)
	}
	polygonBuffer.Buffer = make([]float32, bufferSize)

	bufferOffset := 0
	for _, textureId := range texKeys {
		// The renderer will need the offset and number of floats
		polygonBuffer.MapTextures[textureId].VertOffset = int32(bufferOffset / TexturedVertexSize)
		polygonBuffer.MapTextures[textureId].VertCount = int32(0)

		// Fill in the buffer
		for _, surface := range surfacesByTexture[textureId] {
			polygonBuffer.MapTextures[textureId].VertCount += int32(len(surface.TexturedVertices))

			for _, vertex := range surface.TexturedVertices {
				polygonBuffer.setVertexPosition(bufferOffset, vertex)
				polygonBuffer.setTextureUV(bufferOffset, vertex)
				polygonBuffer.setLightmapUV(bufferOffset, vertex)
				bufferOffset += TexturedVertexSize
			}
		}
	}

	return polygonBuffer
}

func (polygonBuffer *PolygonBuffer) setVertexPosition(bufferOffset int, vertex TexturedVertex) {
	polygonBuffer.Buffer[bufferOffset+0] = vertex.X
	polygonBuffer.Buffer[bufferOffset+1] = vertex.Y
	polygonBuffer.Buffer[bufferOffset+2] = vertex.Z
}

func (polygonBuffer *PolygonBuffer) setTextureUV(bufferOffset int, vertex TexturedVertex) {
	polygonBuffer.Buffer[bufferOffset+3] = vertex.TextureU
	polygonBuffer.Buffer[bufferOffset+4] = vertex.TextureV
}

func (polygonBuffer *PolygonBuffer) setLightmapUV(bufferOffset int, vertex TexturedVertex) {
	polygonBuffer.Buffer[bufferOffset+5] = vertex.LightU
	polygonBuffer.Buffer[bufferOffset+6] = vertex.LightV
}
