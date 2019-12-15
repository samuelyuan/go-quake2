package main

import (
	"./q2file"
	"github.com/go-gl/gl/v4.1-core/gl"
	"math"
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

type LightmapNode struct {
	X      int32
	Y      int32
	Width  int32
	Height int32
	Nodes  []LightmapNode
	Filled bool
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
	lightmap *MapLightmap, // Update lightmap for this face
	faceLightmapOffset uint32,
	mapData *q2file.MapData,
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

	// Check if face has a lightmap
	if texInfo.Flags == 0 {
		lightmapDimensions := getLightmapDimensions(faceVertices, texInfo)
		lightmapRect := readLightmap(lightmap, faceLightmapOffset, lightmapDimensions.Width, lightmapDimensions.Height, mapData)

		// Lightmap texture coordinates
		for i := 0; i < len(surface.TexturedVertices); i++ {
			x := surface.TexturedVertices[i].X
			y := surface.TexturedVertices[i].Y
			z := surface.TexturedVertices[i].Z

			if lightmapRect != nil {
				s := ((x*texInfo.UAxis[0] + y*texInfo.UAxis[1] + z*texInfo.UAxis[2]) + texInfo.UOffset) - lightmapDimensions.MinU
				s += float32((lightmapRect.X * 16) + 8)
				s /= float32(lightmapSize * 16)

				t := ((x*texInfo.VAxis[0] + y*texInfo.VAxis[1] + z*texInfo.VAxis[2]) + texInfo.VOffset) - lightmapDimensions.MinV
				t += float32((lightmapRect.Y * 16) + 8)
				t /= float32(lightmapSize * 16)

				surface.TexturedVertices[i].LightU = s
				surface.TexturedVertices[i].LightV = t
			}
		}
	}
	return surface
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

func readLightmap(lightmap *MapLightmap, offset uint32, width int32, height int32, mapData *q2file.MapData) *LightmapNode {
	if height <= 0 || width <= 0 {
		return nil
	}

	// Navigate lightmap BSP to find correctly sized space
	node := allocateLightmapRect(&lightmap.Root, width, height)
	if node != nil {
		// Each pixel has 4 values for RGBA
		byteCount := width * height * 4
		bytes := make([]uint8, byteCount)
		curByte := 0

		baseIndex := int(offset)
		for i := 0; i < int(width*height); i++ {
			// Change the brightness
			lightScale := 4
			r := int(mapData.LightmapData[baseIndex+0]) * lightScale
			g := int(mapData.LightmapData[baseIndex+1]) * lightScale
			b := int(mapData.LightmapData[baseIndex+2]) * lightScale
			max := 0
			if r > g {
				max = r
			} else {
				max = g
			}
			if b > max {
				max = b
			}
			// Rescale color components if any component exceeds the maximum value (255)
			if max > 255 {
				t := float32(255.0) / float32(max)

				r = int(float32(r) * t)
				g = int(float32(g) * t)
				b = int(float32(b) * t)
			}

			bytes[curByte+0] = uint8(r)
			bytes[curByte+1] = uint8(g)
			bytes[curByte+2] = uint8(b)
			bytes[curByte+3] = 255
			curByte += 4

			// read only 3 components
			baseIndex += 3
		}

		// Copy the lightmap into the allocated rectangle
		gl.BindTexture(gl.TEXTURE_2D, lightmap.Texture)
		gl.TexSubImage2D(gl.TEXTURE_2D, 0, node.X, node.Y, width, height, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(bytes))
	}
	return node
}

// Navigate the Lightmap BSP tree and find an empty spot of the right size
func allocateLightmapRect(node *LightmapNode, width int32, height int32) *LightmapNode {
	// Check child nodes if they exist
	if len(node.Nodes) > 0 {
		newNode := allocateLightmapRect(&node.Nodes[0], width, height)
		if newNode != nil {
			return newNode
		}
		return allocateLightmapRect(&node.Nodes[1], width, height)
	}

	// Already used
	if node.Filled {
		return nil
	}

	// Too small
	if node.Width < width || node.Height < height {
		return nil
	}

	// Allocate if it is a perfect fit
	if node.Width == width && node.Height == height {
		node.Filled = true
		return node
	}

	// Split by width or height
	var nodes []LightmapNode
	if (node.Width - width) > (node.Height - height) {
		nodes = []LightmapNode{
			LightmapNode{
				X:      node.X,
				Y:      node.Y,
				Width:  width,
				Height: node.Height,
				Nodes:  []LightmapNode{},
				Filled: false,
			},
			LightmapNode{
				X:      node.X + width,
				Y:      node.Y,
				Width:  node.Width - width,
				Height: node.Height,
				Nodes:  []LightmapNode{},
				Filled: false,
			},
		}
	} else {
		nodes = []LightmapNode{
			LightmapNode{
				X:      node.X,
				Y:      node.Y,
				Width:  node.Width,
				Height: height,
				Nodes:  []LightmapNode{},
				Filled: false,
			},
			LightmapNode{
				X:      node.X,
				Y:      node.Y + height,
				Width:  node.Width,
				Height: node.Height - height,
				Nodes:  []LightmapNode{},
				Filled: false,
			},
		}
	}
	node.Nodes = nodes
	return allocateLightmapRect(&node.Nodes[0], width, height)
}

func getTextureUV(vtx q2file.Vertex, tex q2file.TexInfo) [2]float32 {
	u := float32(vtx.X*tex.UAxis[0] + vtx.Y*tex.UAxis[1] + vtx.Z*tex.UAxis[2] + tex.UOffset)
	v := float32(vtx.X*tex.VAxis[0] + vtx.Y*tex.VAxis[1] + vtx.Z*tex.VAxis[2] + tex.VOffset)
	return [2]float32{u, v}
}
