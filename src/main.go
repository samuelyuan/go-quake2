package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"sort"
	"strings"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/samuelyuan/go-quake2/q2file"
	"github.com/samuelyuan/go-quake2/render"
)

const (
	windowWidth  = 800
	windowHeight = 600
)

var (
	windowHandler *WindowHandler
)

func createTextureList(
	pakReader io.ReaderAt,
	pakFileMap map[string]q2file.PakFile,
	textureIds map[string]int,
) []render.MapTexture {
	// get sorted strings
	var fileKeys []string
	for texFilename, _ := range textureIds {
		fileKeys = append(fileKeys, texFilename)
	}
	sort.Strings(fileKeys)

	// iterate through filenames in the same order
	oldMapTextures := make([]render.MapTexture, len(fileKeys))
	for i := 0; i < len(fileKeys); i++ {
		// stored in different folder
		// append extension (.wal) as default
		fullFilename := "textures/" + strings.Trim(fileKeys[i], " ") + ".wal"
		fullFilename = strings.ToLower(fullFilename)
		imageData, walData, err := q2file.LoadQ2WALFromPAK(pakReader, pakFileMap, fullFilename)

		if err != nil {
			fmt.Println("Warning: texture", fullFilename, "is missing.")
			index := textureIds[fileKeys[i]]
			oldMapTextures[index] = render.NewMapTexture(0, 0, 0)
			continue

			// log.Fatal("Error loading texture in main:", err)
			// return nil
		}

		// the index is not necessarily in order
		index := textureIds[fileKeys[i]]
		texId := render.BuildWALTexture(imageData, walData)
		oldMapTextures[index] = render.NewMapTexture(texId, walData.Width, walData.Height)
	}

	return oldMapTextures
}

func initMesh(pakFilename string, bspFilename string) (*q2file.MapData, []render.MapTexture, error) {
	pakFile, err := os.Open(pakFilename)
	defer pakFile.Close()

	if err != nil {
		log.Fatal("PAK file ", pakFilename, " doesn't exist")
		return nil, nil, err
	}

	pakFileMap, err := q2file.LoadQ2PAK(pakFile)

	mapData, err := q2file.LoadQ2BSPFromPAK(pakFile, pakFileMap, bspFilename)
	if err != nil {
		log.Fatal("Error loading bsp in main:", err)
		return nil, nil, err
	}
	fmt.Println("BSP map successfully loaded")

	oldMapTextures := createTextureList(pakFile, pakFileMap, mapData.TextureIds)
	if oldMapTextures == nil {
		return nil, nil, fmt.Errorf("Error loading textures")
	}
	fmt.Println("Textures successfully loaded")
	return mapData, oldMapTextures, nil
}

func main() {
	fmt.Println("Starting quake2 bsp loader\n")

	// Run OpenGL code
	runtime.LockOSThread()
	if err := glfw.Init(); err != nil {
		panic(fmt.Errorf("Could not initialize glfw: %v", err))
	}
	defer glfw.Terminate()
	windowHandler = NewWindowHandler(windowWidth, windowHeight, "Quake 2 BSP Loader")

	renderer := render.NewRenderer()
	renderer.Init()

	// Load files
	mapData, oldMapTextures, err := initMesh("./data/pak0.pak", "maps/demo1.bsp")
	if err != nil {
		fmt.Println("Error initializing mesh: ", err)
		return
	}

	bspTree := NewBSPTree(mapData)
	fmt.Println("BSP Tree built")

	camera := NewCamera(windowHandler)
	prevLeaf := -1
	curLeaf := 0

	var renderMap render.RenderMap

	for !windowHandler.shouldClose() {
		windowHandler.startFrame()
		renderer.PrepareFrame(camera.GetViewMatrix(), camera.GetPerspectiveMatrix())

		// Render map data to the screen
		// Figure out which leaf the player is in and only render faces in that leaf
		leaf := bspTree.findLeafNode(0, mapData, camera.GetCameraPosition())
		curLeaf = leaf.LeafIndex
		// Update the polygons if the player is in a different leaf
		if prevLeaf != curLeaf {
			if len(leaf.Faces) > 0 {
				renderMap = render.CreateRenderingData(mapData, oldMapTextures, leaf.Faces)
			}
			prevLeaf = curLeaf
		}
		render.DrawMap(renderMap, renderer.Shader.ProgramShader, renderer.Vao, renderer.Vbo)

		camera.UpdateViewMatrix()
	}
}
