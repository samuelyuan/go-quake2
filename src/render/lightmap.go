package render

import (
	"github.com/go-gl/gl/v4.1-core/gl"
)

var (
	lightmapSize = int32(512)
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
	var texture uint32
	gl.GenTextures(1, &texture)
	gl.BindTexture(gl.TEXTURE_2D, texture)

	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, lightmapSize, lightmapSize, 0, uint32(gl.RGBA), uint32(gl.UNSIGNED_BYTE), nil)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_LINEAR)

	// Set the last pixel to white (for non-lightmapped faces)
	whitePixel := []uint8{255, 255, 255, 255}
	gl.TexSubImage2D(gl.TEXTURE_2D, 0, lightmapSize-1, lightmapSize-1, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(whitePixel))

	// Setup BSP tree here
	return &MapLightmap{
		Texture: texture,
		Root: LightmapNode{
			X:      0,
			Y:      0,
			Width:  lightmapSize,
			Height: lightmapSize,
			Nodes:  []LightmapNode{},
			Filled: false,
		},
	}
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
	return AllocateLightmapRect(&node.Nodes[0], width, height)
}
