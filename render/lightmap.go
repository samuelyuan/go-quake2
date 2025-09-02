package render

import (
	"fmt"
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

func NewLightmap(textureMgr *TextureManager) *MapLightmap {
	textureId, err := textureMgr.CreateLightmapTexture(LIGHTMAP_SIZE)
	if err != nil {
		// For now, we'll use 0 as a fallback, but in production you'd want proper error handling
		textureId = 0
	}

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

func (lightmap *MapLightmap) GenerateMipmaps(textureMgr *TextureManager) {
	textureMgr.GenerateMipmaps(lightmap.Texture)
}

func (lightmap *MapLightmap) CopyMapLightmapToTexture(
	arrayOffset uint32,
	lightmapData []uint8,
	destinationNode *LightmapNode,
	totalPixels int32,
	textureMgr *TextureManager,
) error {
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

	if err := lightmap.updateSubTexture(destinationNode, pixels, textureMgr); err != nil {
		return fmt.Errorf("failed to update subtexture: %w", err)
	}

	return nil
}

func (lightmap *MapLightmap) updateSubTexture(node *LightmapNode, pixels []uint8, textureMgr *TextureManager) error {
	// Copy the lightmap into the allocated rectangle
	return textureMgr.UpdateTextureSubImage(lightmap.Texture, node.X, node.Y, node.Width, node.Height, pixels)
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
