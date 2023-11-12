package render

import (
	"github.com/go-gl/gl/v4.1-core/gl"
)

const (
	LIGHTMAP_SIZE = int32(512)
)

type MapLightmap struct {
	Texture uint32
	Root    LightmapNode
}

type LightmapNode struct {
	X      int32
	Y      int32
	Width  int32
	Height int32
	Nodes  []LightmapNode
	Filled bool
}

func NewLightmap() *MapLightmap {
	textureId := generateTexture()

	// Setup BSP tree here
	return &MapLightmap{
		Texture: textureId,
		Root: LightmapNode{
			X:      0,
			Y:      0,
			Width:  LIGHTMAP_SIZE,
			Height: LIGHTMAP_SIZE,
			Nodes:  []LightmapNode{},
			Filled: false,
		},
	}
}

func generateTexture() uint32 {
	var textureId uint32
	gl.GenTextures(1, &textureId)
	gl.BindTexture(gl.TEXTURE_2D, textureId)

	// Initialize empty region to be updated later
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, LIGHTMAP_SIZE, LIGHTMAP_SIZE, 0, uint32(gl.RGBA), uint32(gl.UNSIGNED_BYTE), nil)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_LINEAR)

	// Set the last pixel to white (for non-lightmapped faces)
	whitePixel := []uint8{255, 255, 255, 255}
	gl.TexSubImage2D(gl.TEXTURE_2D, 0, LIGHTMAP_SIZE-1, LIGHTMAP_SIZE-1, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(whitePixel))

	return textureId
}

func (lightmap *MapLightmap) GenerateMipmaps() {
	gl.BindTexture(gl.TEXTURE_2D, lightmap.Texture)
	gl.GenerateMipmap(gl.TEXTURE_2D)
}

func (lightmap *MapLightmap) CopyMapLightmapToTexture(
	arrayOffset uint32,
	lightmapData []uint8,
	destinationNode *LightmapNode,
	totalPixels int32,
) {
	// Each pixel has 4 values for RGBA
	pixels := make([]uint8, totalPixels*4)
	curByteWrite := 0

	baseIndexRead := int(arrayOffset)
	for i := 0; i < int(totalPixels); i++ {
		// Change the brightness
		lightScale := 4
		r := int(lightmapData[baseIndexRead+0]) * lightScale
		g := int(lightmapData[baseIndexRead+1]) * lightScale
		b := int(lightmapData[baseIndexRead+2]) * lightScale
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

		pixels[curByteWrite+0] = uint8(r)
		pixels[curByteWrite+1] = uint8(g)
		pixels[curByteWrite+2] = uint8(b)
		pixels[curByteWrite+3] = 255
		curByteWrite += 4

		// read only 3 components
		baseIndexRead += 3
	}

	lightmap.updateSubTexture(destinationNode, pixels)
}

func (lightmap *MapLightmap) updateSubTexture(node *LightmapNode, pixels []uint8) {
	// Copy the lightmap into the allocated rectangle
	gl.BindTexture(gl.TEXTURE_2D, lightmap.Texture)
	gl.TexSubImage2D(gl.TEXTURE_2D, 0, node.X, node.Y, node.Width, node.Height, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(pixels))
}

// Navigate the Lightmap BSP tree and find an empty spot of the right size
func AllocateLightmapRect(node *LightmapNode, width int32, height int32) *LightmapNode {
	// Check child nodes if they exist
	if len(node.Nodes) > 0 {
		newNode := AllocateLightmapRect(&node.Nodes[0], width, height)
		if newNode != nil {
			return newNode
		}
		return AllocateLightmapRect(&node.Nodes[1], width, height)
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
			{
				X:      node.X,
				Y:      node.Y,
				Width:  width,
				Height: node.Height,
				Nodes:  []LightmapNode{},
				Filled: false,
			},
			{
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
			{
				X:      node.X,
				Y:      node.Y,
				Width:  node.Width,
				Height: height,
				Nodes:  []LightmapNode{},
				Filled: false,
			},
			{
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
	return AllocateLightmapRect(&node.Nodes[0], width, height)
}
