package render

import (
	"fmt"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/samuelyuan/go-quake2/q2file"
)

// TextureManager handles all OpenGL texture operations
type TextureManager struct {
	textures map[string]uint32
}

// NewTextureManager creates a new texture manager
func NewTextureManager() *TextureManager {
	return &TextureManager{
		textures: make(map[string]uint32),
	}
}

// CreateTexture2D creates a 2D texture with the given parameters
func (tm *TextureManager) CreateTexture2D(width, height int32, format uint32, data []uint8) (uint32, error) {
	// Input validation
	if width <= 0 || height <= 0 {
		return 0, fmt.Errorf("invalid texture dimensions: %dx%d", width, height)
	}

	// Validate format
	if format != gl.RGB && format != gl.RGBA && format != gl.RED && format != gl.GREEN && format != gl.BLUE {
		return 0, fmt.Errorf("unsupported texture format: %d", format)
	}

	// Validate data size if provided
	if data != nil {
		expectedSize := width * height * getFormatChannels(format)
		if int32(len(data)) < expectedSize {
			return 0, fmt.Errorf("insufficient data for texture size: expected %d bytes, got %d", expectedSize, len(data))
		}
	}

	var textureId uint32
	gl.GenTextures(1, &textureId)

	gl.BindTexture(gl.TEXTURE_2D, textureId)
	gl.TexImage2D(gl.TEXTURE_2D, 0, int32(format), width, height, 0, format, gl.UNSIGNED_BYTE, gl.Ptr(data))

	// Set default parameters
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, DEFAULT_TEXTURE_MAG_FILTER)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, DEFAULT_TEXTURE_MIN_FILTER)

	return textureId, nil
}

// CreateLightmapTexture creates a lightmap texture with appropriate parameters
func (tm *TextureManager) CreateLightmapTexture(size int32) (uint32, error) {
	// Input validation
	if size <= 0 {
		return 0, fmt.Errorf("invalid lightmap size: %d", size)
	}

	// Check if size is a power of 2 (recommended for OpenGL)
	if size&(size-1) != 0 {
		return 0, fmt.Errorf("lightmap size should be a power of 2, got: %d", size)
	}

	var textureId uint32
	gl.GenTextures(1, &textureId)

	gl.BindTexture(gl.TEXTURE_2D, textureId)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, size, size, 0, gl.RGBA, gl.UNSIGNED_BYTE, nil)

	// Lightmap-specific parameters
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, DEFAULT_TEXTURE_MAG_FILTER)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, LIGHTMAP_TEXTURE_MIN_FILTER)

	return textureId, nil
}

// CreateWALTexture creates a texture from WAL image data
func (tm *TextureManager) CreateWALTexture(imageData []uint8, walData q2file.WalHeader) (uint32, error) {
	textureId, err := tm.CreateTexture2D(int32(walData.Width), int32(walData.Height), gl.RGB, imageData)
	if err != nil {
		return 0, fmt.Errorf("failed to create WAL texture: %w", err)
	}

	// Store texture reference
	textureName := fmt.Sprintf("wal_%dx%d", walData.Width, walData.Height)
	tm.textures[textureName] = textureId

	return textureId, nil
}

// UpdateTextureSubImage updates a portion of a texture
func (tm *TextureManager) UpdateTextureSubImage(textureId uint32, x, y, width, height int32, data []uint8) error {
	// Input validation
	if textureId == 0 {
		return fmt.Errorf("invalid texture ID: 0")
	}
	if x < 0 || y < 0 || width <= 0 || height <= 0 {
		return fmt.Errorf("invalid subimage parameters: x=%d, y=%d, width=%d, height=%d", x, y, width, height)
	}
	if data == nil {
		return fmt.Errorf("data cannot be nil")
	}

	// Check data size (RGBA = 4 bytes per pixel)
	expectedSize := width * height * 4
	if int32(len(data)) < expectedSize {
		return fmt.Errorf("insufficient data for subimage: expected %d bytes, got %d", expectedSize, len(data))
	}

	gl.BindTexture(gl.TEXTURE_2D, textureId)
	gl.TexSubImage2D(gl.TEXTURE_2D, 0, x, y, width, height, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(data))

	return nil
}

// GenerateMipmaps generates mipmaps for a texture
func (tm *TextureManager) GenerateMipmaps(textureId uint32) {
	gl.BindTexture(gl.TEXTURE_2D, textureId)
	gl.GenerateMipmap(gl.TEXTURE_2D)
}

// DeleteTexture deletes a texture and frees OpenGL resources
func (tm *TextureManager) DeleteTexture(textureId uint32) {
	gl.DeleteTextures(1, &textureId)
}

// Cleanup frees all OpenGL texture resources
func (tm *TextureManager) Cleanup() {
	for _, textureId := range tm.textures {
		tm.DeleteTexture(textureId)
	}
	tm.textures = make(map[string]uint32)
}

// getFormatChannels returns the number of color channels for a given format
func getFormatChannels(format uint32) int32 {
	switch format {
	case gl.RGB:
		return 3
	case gl.RGBA:
		return 4
	case gl.RED, gl.GREEN, gl.BLUE:
		return 1
	default:
		return 1 // fallback
	}
}
