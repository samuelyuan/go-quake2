package render

import (
	"math"

	"github.com/samuelyuan/go-quake2/q2file"
)

// Contains all the triangles of a face to be passed to the renderer
type Surface struct {
	TexInfo          q2file.TexInfo
	TexturedVertices []TexturedVertex
}

type TexturedVertex struct {
	// Position coordinates
	X float32
	Y float32
	Z float32

	// Texture coordinates
	TextureU float32
	TextureV float32

	// Lightmap coordinates
	LightU float32
	LightV float32
}

type LightmapDimensions struct {
	Width  int32
	Height int32
	MinU   float32
	MinV   float32
}

func NewSurface(
	faceVertices []q2file.Vertex,
	texInfo q2file.TexInfo,
	textureWidth uint32,
	textureHeight uint32,
) *Surface {
	surface := &Surface{}
	surface.TexInfo = texInfo
	surface.TexturedVertices = make([]TexturedVertex, len(faceVertices))
	for i := 0; i < len(faceVertices); i++ {
		texturedVertex := TexturedVertex{}

		x := faceVertices[i].X
		y := faceVertices[i].Y
		z := faceVertices[i].Z
		texturedVertex.X = x
		texturedVertex.Y = y
		texturedVertex.Z = z

		uv := getTextureUV(faceVertices[i], texInfo)
		texturedVertex.TextureU = uv[0] / float32(textureWidth)
		texturedVertex.TextureV = uv[1] / float32(textureHeight)

		texturedVertex.LightU = 0.999
		texturedVertex.LightV = 0.999
		surface.TexturedVertices[i] = texturedVertex
	}

	return surface
}

func (surface *Surface) UpdateLightmap(
	lightmap *MapLightmap, // Update lightmap for this face
	faceVertices []q2file.Vertex,
	texInfo q2file.TexInfo,
	faceLightmapOffset uint32,
	mapData *q2file.MapData,
) {
	// Check if face has a lightmap
	if texInfo.Flags == 0 {
		lightmapDimensions := getLightmapDimensions(faceVertices, texInfo)
		if lightmapDimensions.Height <= 0 || lightmapDimensions.Width <= 0 {
			return
		}

		// Navigate lightmap BSP to find correctly sized space
		lightmapRect := AllocateLightmapRect(&lightmap.Root, lightmapDimensions.Width, lightmapDimensions.Height)
		if lightmapRect == nil {
			return
		}

		totalPixels := lightmapDimensions.Width * lightmapDimensions.Height
		lightmap.CopyMapLightmapToTexture(faceLightmapOffset, mapData.LightmapData, lightmapRect, totalPixels)

		// Update lightmap texture coordinates for rendering
		for i := 0; i < len(surface.TexturedVertices); i++ {
			x := surface.TexturedVertices[i].X
			y := surface.TexturedVertices[i].Y
			z := surface.TexturedVertices[i].Z

			s := ((x*texInfo.UAxis[0] + y*texInfo.UAxis[1] + z*texInfo.UAxis[2]) + texInfo.UOffset) - lightmapDimensions.MinU
			s += float32((lightmapRect.X * 16) + 8)
			s /= float32(LIGHTMAP_SIZE * 16)

			t := ((x*texInfo.VAxis[0] + y*texInfo.VAxis[1] + z*texInfo.VAxis[2]) + texInfo.VOffset) - lightmapDimensions.MinV
			t += float32((lightmapRect.Y * 16) + 8)
			t /= float32(LIGHTMAP_SIZE * 16)

			surface.TexturedVertices[i].LightU = s
			surface.TexturedVertices[i].LightV = t
		}
	}
}

// Get the width and height of the lightmap
func getLightmapDimensions(faceVertices []q2file.Vertex, texInfo q2file.TexInfo) LightmapDimensions {
	startUV := getTextureUV(faceVertices[0], texInfo)

	// Find the Min and Max UV's for a face
	startUV0 := float64(startUV[0])
	startUV1 := float64(startUV[1])
	minU := math.Floor(startUV0)
	minV := math.Floor(startUV1)
	maxU := math.Floor(startUV0)
	maxV := math.Floor(startUV1)

	for i := 1; i < len(faceVertices); i++ {
		uv := getTextureUV(faceVertices[i], texInfo)
		uv0 := float64(uv[0])
		uv1 := float64(uv[1])

		if math.Floor(uv0) < minU {
			minU = math.Floor(uv0)
		}
		if math.Floor(uv1) < minV {
			minV = math.Floor(uv1)
		}
		if math.Floor(uv0) > maxU {
			maxU = math.Floor(uv0)
		}
		if math.Floor(uv1) > maxV {
			maxV = math.Floor(uv1)
		}
	}

	// Calculate the lightmap dimensions
	return LightmapDimensions{
		Width:  int32(math.Ceil(maxU/16) - math.Floor(minU/16) + 1),
		Height: int32(math.Ceil(maxV/16) - math.Floor(minV/16) + 1),
		MinU:   float32(math.Floor(minU)),
		MinV:   float32(math.Floor(minV)),
	}
}

func getTextureUV(vtx q2file.Vertex, tex q2file.TexInfo) [2]float32 {
	u := float32(vtx.X*tex.UAxis[0] + vtx.Y*tex.UAxis[1] + vtx.Z*tex.UAxis[2] + tex.UOffset)
	v := float32(vtx.X*tex.VAxis[0] + vtx.Y*tex.VAxis[1] + vtx.Z*tex.VAxis[2] + tex.VOffset)
	return [2]float32{u, v}
}
