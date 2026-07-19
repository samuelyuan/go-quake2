package render

import (
	"github.com/samuelyuan/go-quake2/q2file"
)

const (
	SURFACE_SKY = uint32(4)
)

type RenderMap struct {
	MapTextures  []MapTexture
	MapLightmap  *MapLightmap
	VertexBuffer []float32
}

// CreateRenderingData processes map data and creates rendering structures
func CreateRenderingData(mapData *q2file.MapData, mapTextures []MapTexture, faceIds []int, textureMgr *TextureManager) RenderMap {
	surfacesByTexture := make(map[int][]Surface)

	// lightmap is shared by all polygons
	lightmap := NewLightmap(textureMgr)

	for _, faceId := range faceIds {
		faceInfo := mapData.Faces[faceId]
		texInfo := mapData.TexInfos[faceInfo.TextureInfo]

		// Hide skybox
		if texInfo.Flags&SURFACE_SKY != 0 {
			continue
		}

		// Get index in texture array
		filename := q2file.BytesToString(texInfo.TextureName[:])
		texId := mapData.TextureIds[filename]
		mapTexture := mapTextures[texId]

		// Check if there are any surfaces mapped to this texture
		_, ok := surfacesByTexture[texId]
		if !ok {
			surfacesByTexture[texId] = make([]Surface, 0)
		}

		faceVertices := getAllFaceVertices(mapData, faceInfo)
		surface := NewSurface(faceVertices, texInfo, mapTexture.Width, mapTexture.Height)
		surface.UpdateLightmap(lightmap, faceVertices, texInfo, faceInfo.LightmapOffset, mapData, textureMgr)

		// Add all triangle data for this texture
		surfacesByTexture[texId] = append(surfacesByTexture[texId], *surface)
	}

	lightmap.GenerateMipmaps(textureMgr)

	polygonBuffer := NewPolygonBuffer(surfacesByTexture, mapTextures)
	renderMap := RenderMap{
		MapLightmap:  lightmap,
		MapTextures:  polygonBuffer.MapTextures,
		VertexBuffer: polygonBuffer.Buffer,
	}
	return renderMap
}

// getAllFaceVertices extracts all vertices for a face using triangle fan generation
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

// getEdgeVertex extracts a vertex from an edge, handling positive/negative edge indices
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
