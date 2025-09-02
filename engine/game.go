package engine

import (
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/samuelyuan/go-quake2/client"
	"github.com/samuelyuan/go-quake2/q2file"
	"github.com/samuelyuan/go-quake2/render"
)

// Configuration constants
const (
	// Window configuration
	DefaultWindowWidth  = 800
	DefaultWindowHeight = 600
	DefaultWindowTitle  = "Quake 2 BSP Loader"

	// Camera configuration
	MouseSensitivity = 0.7
	CameraSpeed      = 200.0
	CameraFOV        = 45.0
	NearPlane        = 0.1
	FarPlane         = 4096.0

	// BSP configuration
	ClusterInvalidID = 65535

	// File paths
	DefaultPakFile   = "./data/pak0.pak"
	DefaultMapFile   = "maps/demo1.bsp"
	TexturePath      = "textures/"
	TextureExtension = ".wal"
)

// Game represents the main game engine
type Game struct {
	windowHandler *client.WindowHandler
	renderer      *render.Renderer
	mapData       *q2file.MapData
	mapTextures   []render.MapTexture
	bspTree       *BSPTree
	camera        *Camera
	prevLeaf      int
	curLeaf       int
	renderMap     render.RenderMap
	skybox        *render.Skybox

	// Simple performance tracking
	frameCount    int
	lastFPSUpdate time.Time
}

// NewGame creates and initializes a new game instance
func NewGame() (*Game, error) {
	game := &Game{
		prevLeaf: -1,
		curLeaf:  0,
	}

	// Initialize window handler
	game.windowHandler = client.NewWindowHandler(
		DefaultWindowWidth,
		DefaultWindowHeight,
		DefaultWindowTitle,
	)

	// Initialize renderer
	game.renderer = render.NewRenderer()
	game.renderer.Init()

	// Initialize camera
	game.camera = NewCamera(game.windowHandler)

	// Load game data
	if err := game.loadGameData(); err != nil {
		return nil, fmt.Errorf("failed to load game data: %w", err)
	}

	return game, nil
}

// loadGameData loads the BSP map and textures
func (g *Game) loadGameData() error {
	var err error

	g.mapData, g.mapTextures, err = g.initMesh(DefaultPakFile, DefaultMapFile)
	if err != nil {
		return fmt.Errorf("failed to load game data: %w", err)
	}

	g.bspTree = NewBSPTree(g.mapData)
	log.Println("BSP Tree built successfully")

	// Create a simple skybox (for now, just a solid color)
	g.skybox = render.NewSkybox(0) // 0 = no texture, will use solid color
	g.renderer.SetSkybox(g.skybox)
	log.Println("Skybox initialized successfully")

	return nil
}

// Run starts the main game loop
func (g *Game) Run() error {
	log.Println("Starting game loop...")

	for !g.windowHandler.ShouldClose() {
		if err := g.update(); err != nil {
			return fmt.Errorf("game update failed: %w", err)
		}
	}

	return nil
}

// update performs one frame update
func (g *Game) update() error {
	g.windowHandler.StartFrame()

	// Draw skybox background FIRST (simple clear to sky blue)
	g.renderer.DrawSkybox(g.camera.GetViewMatrix(), g.camera.GetPerspectiveMatrix())

	g.renderer.PrepareFrame(g.camera.GetViewMatrix(), g.camera.GetPerspectiveMatrix())

	// Update camera and find current leaf
	g.camera.UpdateViewMatrix()
	leaf := g.bspTree.findLeafNode(0, g.mapData, g.camera.GetCameraPosition())
	g.curLeaf = leaf.LeafIndex

	// Update rendering data if leaf changed
	if g.prevLeaf != g.curLeaf {
		if len(leaf.Faces) > 0 {
			g.renderMap = render.CreateRenderingData(g.mapData, g.mapTextures, leaf.Faces, g.renderer.GetTextureManager())
		}
		g.prevLeaf = g.curLeaf
	}

	// Render the map
	render.DrawMap(g.renderer, g.renderMap)

	// Simple FPS tracking
	g.frameCount++
	if time.Since(g.lastFPSUpdate) >= time.Second {
		fps := float64(g.frameCount) / time.Since(g.lastFPSUpdate).Seconds()
		log.Printf("FPS: %.1f", fps)
		g.frameCount = 0
		g.lastFPSUpdate = time.Now()
	}

	return nil
}

// initMesh loads the BSP map and textures from PAK files
func (g *Game) initMesh(pakFilename, bspFilename string) (*q2file.MapData, []render.MapTexture, error) {
	pakFile, err := os.Open(pakFilename)
	if err != nil {
		return nil, nil, fmt.Errorf("PAK file %s doesn't exist: %w", pakFilename, err)
	}
	defer pakFile.Close()

	pakFileMap, err := q2file.LoadQ2PAK(pakFile)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load PAK file: %w", err)
	}

	mapData, err := q2file.LoadQ2BSPFromPAK(pakFile, pakFileMap, bspFilename)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load BSP file: %w", err)
	}
	log.Println("BSP map successfully loaded")

	mapTextures := g.createTextureList(pakFile, pakFileMap, mapData.TextureIds)
	if mapTextures == nil {
		return nil, nil, fmt.Errorf("failed to load textures")
	}
	log.Println("Textures successfully loaded")

	return mapData, mapTextures, nil
}

// createTextureList creates a list of map textures from the PAK file
func (g *Game) createTextureList(
	pakReader io.ReaderAt,
	pakFileMap map[string]q2file.PakFile,
	textureIds map[string]int,
) []render.MapTexture {
	// Get sorted texture filenames for consistent loading
	var fileKeys []string
	for texFilename := range textureIds {
		fileKeys = append(fileKeys, texFilename)
	}
	sort.Strings(fileKeys)

	// Create texture list
	mapTextures := make([]render.MapTexture, len(fileKeys))
	for _, filename := range fileKeys {
		// Precompute full texture path once (not every frame)
		fullFilename := TexturePath + strings.Trim(filename, " ") + TextureExtension
		fullFilename = strings.ToLower(fullFilename)

		// Load texture data
		imageData, walData, err := q2file.LoadQ2WALFromPAK(pakReader, pakFileMap, fullFilename)
		if err != nil {
			log.Printf("Warning: texture %s is missing: %v", fullFilename, err)
			index := textureIds[filename]
			mapTextures[index] = render.NewMapTexture(0, 0, 0)
			continue
		}

		// Delegate texture creation to render package
		index := textureIds[filename]
		texID, err := g.renderer.GetTextureManager().CreateWALTexture(imageData, walData)
		if err != nil {
			log.Printf("Warning: failed to create texture %s: %v", fullFilename, err)
			texID = 0 // Use fallback texture ID
		}
		mapTextures[index] = render.NewMapTexture(texID, walData.Width, walData.Height)
	}

	return mapTextures
}
