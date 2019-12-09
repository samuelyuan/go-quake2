package main

import (
	"./q2file"
	"fmt"
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"io"
	"log"
	"os"
	"runtime"
	"sort"
	"strings"
)

const (
	windowWidth  = 800
	windowHeight = 600
)

var (
	lightmapSize = int32(512)

	windowHandler *WindowHandler
)

type RenderMap struct {
	MapTextures []MapTexture
	MapLightmap *MapLightmap
}

type MapTexture struct {
	Id         uint32
	Width      uint32
	Height     uint32
	VertOffset int32
	VertCount  int32
}

type MapLightmap struct {
	Texture uint32
	Root    LightmapNode
}

func initOpenGL() uint32 {
	if err := gl.Init(); err != nil {
		panic(err)
	}

	version := gl.GoStr(gl.GetString(gl.VERSION))
	fmt.Println("OpenGL version", version)

	shader := q2file.NewShader()
	return shader.ProgramShader
}

// Initialize texture in OpenGL using image data
func buildTexture(imageData []uint8, walData q2file.WalHeader) uint32 {
	var texId uint32
	gl.GenTextures(1, &texId)
	gl.BindTexture(gl.TEXTURE_2D, texId)

	// Give the image to OpenGL
	gl.TexImage2D(uint32(gl.TEXTURE_2D), 0, int32(gl.RGB), int32(walData.Width), int32(walData.Height),
		0, uint32(gl.RGB), uint32(gl.UNSIGNED_BYTE), gl.Ptr(imageData))

	// Set texture wrapping/filtering options
	gl.TexParameteri(uint32(gl.TEXTURE_2D), gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(uint32(gl.TEXTURE_2D), gl.TEXTURE_MIN_FILTER, gl.LINEAR)

	return texId
}

func drawMap(vertices []float32, renderMap RenderMap, programShader uint32) {
	var vao uint32
	var vbo uint32

	// Create buffers/arrays
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)

	// Load data
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	floatSize := 4 // size of float32 is 4
	// Fill vertex buffer
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*floatSize, gl.Ptr(vertices), gl.STATIC_DRAW)

	// 3 floats for vertex, 2 floats for texture UV, 2 floats for lightmap UV
	stride := int32(TexturedVertexSize * floatSize)

	// Position attribute
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, stride, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)

	// Texture
	gl.VertexAttribPointer(1, 2, gl.FLOAT, false, stride, gl.PtrOffset(3*floatSize))
	gl.EnableVertexAttribArray(1)

	// Lightmap
	gl.VertexAttribPointer(2, 2, gl.FLOAT, false, stride, gl.PtrOffset(5*floatSize))
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

func createTextureList(pakReader io.ReaderAt, pakFileMap map[string]q2file.PakFile, textureIds map[string]int) []MapTexture {
	// get sorted strings
	var fileKeys []string
	for texFilename, _ := range textureIds {
		fileKeys = append(fileKeys, texFilename)
	}
	sort.Strings(fileKeys)

	// iterate through filenames in the same order
	oldMapTextures := make([]MapTexture, len(fileKeys))
	for i := 0; i < len(fileKeys); i++ {
		// stored in different folder
		// append extension (.wal) as default
		fullFilename := "textures/" + strings.Trim(fileKeys[i], " ") + ".wal"
		imageData, walData, err := q2file.LoadQ2WALFromPAK(pakReader, pakFileMap, fullFilename)
		if err != nil {
			log.Fatal("Error loading texture in main:", err)
			return nil
		}

		// Initialize texture
		texId := buildTexture(imageData, walData)

		// the index is not necessarily in order
		index := textureIds[fileKeys[i]]
		oldMapTextures[index] = MapTexture{}
		oldMapTextures[index].Width = walData.Width
		oldMapTextures[index].Height = walData.Height
		// opengl texture id
		oldMapTextures[index].Id = texId
	}

	return oldMapTextures
}

func createRenderingData(mapData *q2file.MapData, mapTextures []MapTexture) ([]float32, RenderMap) {
	vertsByTexture := make(map[int][]Surface)

	lightmap := NewLightmap()

	var offset uint16
	for faceIdx := 0; faceIdx < len(mapData.Faces); faceIdx++ {
		faceInfo := mapData.Faces[faceIdx]
		texInfo := mapData.TexInfos[faceInfo.TextureInfo]

		// Hide skybox
		if texInfo.Flags&4 != 0 {
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

		// Generate triangle fan from map face
		var faceVertices []q2file.Vertex
		// Fix the first vertex
		v0 := getEdgeVertex(mapData, int(faceInfo.FirstEdge))
		v1 := getEdgeVertex(mapData, int(faceInfo.FirstEdge)+1)

		for offset = 2; offset < faceInfo.NumEdges; offset++ {
			v2 := getEdgeVertex(mapData, int(faceInfo.FirstEdge)+int(offset))

			// Add triangle
			faceVertices = append(faceVertices, v0, v1, v2)

			// Move to the next triangle
			v1 = v2
		}

		surface := NewSurface(faceVertices, texInfo, mapTexture.Width, mapTexture.Height, lightmap, faceInfo.LightmapOffset, mapData)

		// Add all triangle data for this texture
		vertsByTexture[texId] = append(vertsByTexture[texId], *surface)
	}

	// Generate mipmaps for the lightmap
	gl.BindTexture(gl.TEXTURE_2D, lightmap.Texture)
	gl.GenerateMipmap(gl.TEXTURE_2D)

	polygonBuffer := NewPolygonBuffer(vertsByTexture, mapTextures)
	renderMap := RenderMap{
		MapLightmap: lightmap,
		MapTextures: polygonBuffer.MapTextures,
	}
	return polygonBuffer.Buffer, renderMap
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

func initMesh(pakFilename string, bspFilename string) (*q2file.MapData, []MapTexture, error) {
	pakFile, err := os.Open(pakFilename)
	defer pakFile.Close()

	if err != nil {
		log.Fatal("PAK file doesn't exist")
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
	programShader := initOpenGL()

	gl.ClearColor(0.0, 0.0, 0.0, 1.0)
	gl.ClearDepth(1.0)
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LEQUAL)

	// Set appropriate blending mode
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	gl.Enable(gl.CULL_FACE)
	gl.CullFace(gl.FRONT)

	// Load files
	mapData, oldMapTextures, err := initMesh("./data/pak0.pak", "maps/demo1.bsp")
	if err != nil {
		fmt.Println("Error initializing mesh: ", err)
		return
	}

	vertexBuffer, renderMap := createRenderingData(mapData, oldMapTextures)
	fmt.Println("Rendering data is generated. Begin rendering.")

	camera := NewCamera(windowHandler)
	for !windowHandler.shouldClose() {
		windowHandler.startFrame()

		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		// Activate shader
		gl.UseProgram(programShader)

		// Create transformations
		view := camera.GetViewMatrix()
		projection := camera.GetPerspectiveMatrix()

		// Get their uniform location
		viewLoc := gl.GetUniformLocation(programShader, gl.Str("view\x00"))
		projectionLoc := gl.GetUniformLocation(programShader, gl.Str("projection\x00"))

		// Pass the matrices to the shader
		gl.UniformMatrix4fv(viewLoc, 1, false, &view[0])
		gl.UniformMatrix4fv(projectionLoc, 1, false, &projection[0])

		// Render map data to the screen
		drawMap(vertexBuffer, renderMap, programShader)

		camera.UpdateViewMatrix()
	}
}
