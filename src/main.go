package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"sort"
	"strings"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"go-quake2/q2file"
	"go-quake2/render"
)

const (
	windowWidth  = 800
	windowHeight = 600
)

var (
	SurfaceSky = uint32(4)
	floatSize  = 4

	windowHandler *WindowHandler
)

type RenderMap struct {
	MapTextures []render.MapTexture
	MapLightmap *render.MapLightmap
}

func initOpenGL() uint32 {
	if err := gl.Init(); err != nil {
		panic(err)
	}

	version := gl.GoStr(gl.GetString(gl.VERSION))
	fmt.Println("OpenGL version", version)

	shader := render.NewShader("render/goquake2.vert", "render/goquake2.frag")
	return shader.ProgramShader
}

func drawMap(vertices []float32, renderMap RenderMap, programShader uint32, vao uint32, vbo uint32) {
	gl.BindVertexArray(vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)

	// Fill vertex buffer
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*floatSize, gl.Ptr(vertices), gl.STATIC_DRAW)

	// 3 floats for vertex, 2 floats for texture UV, 2 floats for lightmap UV
	stride := int32(render.TexturedVertexSize * floatSize)

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

func createTextureList(
	pakReader io.ReaderAt,
	pakFileMap map[string]q2file.PakFile,
	textureIds map[string]int) []render.MapTexture {
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

func createRenderingData(mapData *q2file.MapData, mapTextures []render.MapTexture, faceIds []int) ([]float32, RenderMap) {
	vertsByTexture := make(map[int][]render.Surface)

	lightmap := render.NewLightmap()

	var offset uint16
	allSurfaces := make([]render.Surface, 0)
	for _, faceId := range faceIds {
		faceInfo := mapData.Faces[faceId]
		texInfo := mapData.TexInfos[faceInfo.TextureInfo]

		// Hide skybox
		if texInfo.Flags&SurfaceSky != 0 {
			continue
		}

		// Get index in texture array
		filename := getTextureFilename(texInfo)
		texId := mapData.TextureIds[filename]
		mapTexture := mapTextures[texId]

		_, ok := vertsByTexture[texId]
		if !ok {
			vertsByTexture[texId] = make([]render.Surface, 0)
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

		surface := render.NewSurface(faceVertices, texInfo, mapTexture.Width, mapTexture.Height, lightmap, faceInfo.LightmapOffset, mapData)

		// Add all triangle data for this texture
		vertsByTexture[texId] = append(vertsByTexture[texId], *surface)
		allSurfaces = append(allSurfaces, *surface)
	}

	// Generate mipmaps for the lightmap
	gl.BindTexture(gl.TEXTURE_2D, lightmap.Texture)
	gl.GenerateMipmap(gl.TEXTURE_2D)

	polygonBuffer := render.NewPolygonBuffer(vertsByTexture, mapTextures)
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
	programShader := initOpenGL()

	gl.ClearColor(0.0, 0.0, 0.0, 1.0)
	gl.Enable(gl.DEPTH_TEST)

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

	bspTree := NewBSPTree(mapData)
	fmt.Println("BSP Tree built")

	allFaceIds := make([]int, len(mapData.Faces))
	for faceIdx := 0; faceIdx < len(mapData.Faces); faceIdx++ {
		allFaceIds[faceIdx] = faceIdx
	}
	vertexBuffer, renderMap := createRenderingData(mapData, oldMapTextures, allFaceIds)
	fmt.Println("Rendering data is generated. Begin rendering.")

	camera := NewCamera(windowHandler)
	prevLeaf := -1
	curLeaf := 0

	// Create buffers/arrays
	var vao uint32
	var vbo uint32
	gl.GenVertexArrays(1, &vao)
	gl.GenBuffers(1, &vbo)

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
		// Figure out which leaf the player is in and only render faces in that leaf
		leaf := bspTree.findLeafNode(0, mapData, camera.GetCameraPosition())
		curLeaf = leaf.LeafIndex
		// Update the polygons if the player is in a different leaf
		if prevLeaf != curLeaf {
			if len(leaf.Faces) > 0 {
				vertexBuffer, renderMap = createRenderingData(mapData, oldMapTextures, leaf.Faces)
			}
			prevLeaf = curLeaf
		}
		drawMap(vertexBuffer, renderMap, programShader, vao, vbo)

		camera.UpdateViewMatrix()
	}
}
