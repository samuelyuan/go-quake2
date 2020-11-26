package render

import (
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/samuelyuan/go-quake2/q2file"
)

const (
	SURFACE_SKY = uint32(4)
	FLOAT_SIZE  = 4
)

type RenderMap struct {
	MapTextures  []MapTexture
	MapLightmap  *MapLightmap
	VertexBuffer []float32
}

func CreateRenderingData(mapData *q2file.MapData, mapTextures []MapTexture, faceIds []int) RenderMap {
	vertsByTexture := make(map[int][]Surface)

	// lightmap is shared by all polygons
	lightmap := NewLightmap()

	allSurfaces := make([]Surface, 0)
	for _, faceId := range faceIds {
		faceInfo := mapData.Faces[faceId]
		texInfo := mapData.TexInfos[faceInfo.TextureInfo]

		// Hide skybox
		if texInfo.Flags&SURFACE_SKY != 0 {
			continue
		}

		// Get index in texture array
		filename := getTextureFilename(texInfo)
		texId := mapData.TextureIds[filename]
		mapTexture := mapTextures[texId]

		_, ok := vertsByTexture[texId]
		if !ok {
			vertsByTexture[texId] = make([]Surface, 0)
		}

		faceVertices := getAllFaceVertices(mapData, faceInfo)
		surface := NewSurface(faceVertices, texInfo, mapTexture.Width, mapTexture.Height)
		surface.UpdateLightmap(lightmap, faceVertices, texInfo, faceInfo.LightmapOffset, mapData)

		// Add all triangle data for this texture
		vertsByTexture[texId] = append(vertsByTexture[texId], *surface)
		allSurfaces = append(allSurfaces, *surface)
	}

	lightmap.GenerateMipmaps()

	polygonBuffer := NewPolygonBuffer(vertsByTexture, mapTextures)
	renderMap := RenderMap{
		MapLightmap:  lightmap,
		MapTextures:  polygonBuffer.MapTextures,
		VertexBuffer: polygonBuffer.Buffer,
	}
	return renderMap
}

func DrawMap(renderMap RenderMap, programShader uint32, vao uint32, vbo uint32) {
	gl.BindVertexArray(vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)

	vertices := renderMap.VertexBuffer

	// Fill vertex buffer
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*FLOAT_SIZE, gl.Ptr(vertices), gl.STATIC_DRAW)

	// 3 floats for vertex, 2 floats for texture UV, 2 floats for lightmap UV
	stride := int32(TexturedVertexSize * FLOAT_SIZE)

	// Position attribute
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, stride, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)

	// Texture
	gl.VertexAttribPointer(1, 2, gl.FLOAT, false, stride, gl.PtrOffset(3*FLOAT_SIZE))
	gl.EnableVertexAttribArray(1)

	// Lightmap
	gl.VertexAttribPointer(2, 2, gl.FLOAT, false, stride, gl.PtrOffset(5*FLOAT_SIZE))
	gl.EnableVertexAttribArray(2)

	diffuseUniform := gl.GetUniformLocation(programShader, gl.Str("diffuse\x00"))
	gl.Uniform1i(diffuseUniform, 0)

	// Bind the lightmap texture shared by all the faces
	gl.ActiveTexture(gl.TEXTURE1)
	gl.BindTexture(gl.TEXTURE_2D, (*renderMap.MapLightmap).Texture)
	lightmapUniform := gl.GetUniformLocation(programShader, gl.Str("lightmap\x00"))
	gl.Uniform1i(lightmapUniform, 1)

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

func getAllFaceVertices(mapData *q2file.MapData, faceInfo q2file.Face) []q2file.Vertex {
	faceVertices := make([]q2file.Vertex, 0)

	// Fix the first vertex
	v0 := getEdgeVertex(mapData, int(faceInfo.FirstEdge))
	v1 := getEdgeVertex(mapData, int(faceInfo.FirstEdge)+1)

	// Generate triangle fan from map face
	var offset uint16
	for offset = 2; offset < faceInfo.NumEdges; offset++ {
		v2 := getEdgeVertex(mapData, int(faceInfo.FirstEdge)+int(offset))

		// Add triangle
		faceVertices = append(faceVertices, v0, v1, v2)

		// Move to the next triangle
		v1 = v2
	}

	return faceVertices
}

func getEdgeVertex(mapData *q2file.MapData, faceEdgeIdx int) q2file.Vertex {
	edgeIdx := int(mapData.FaceEdges[faceEdgeIdx].EdgeIndex)

	// Edge index is positive
	if edgeIdx >= 0 {
		// Return first vertex as the start of the edge
		return mapData.Vertices[mapData.Edges[edgeIdx].V1]
	}

	// Edge index is negative
	// Return second vertex as the start of the edge
	return mapData.Vertices[mapData.Edges[-edgeIdx].V2]
}

func getTextureFilename(texInfo q2file.TexInfo) string {
	// convert filename byte array to string
	filename := ""
	for i := 0; i < len(texInfo.TextureName); i++ {
		// end of string
		if texInfo.TextureName[i] == 0 {
			break
		}
		filename += string(texInfo.TextureName[i])
	}
	return filename
}
