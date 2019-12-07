package main

import (
	"./render"
	"fmt"
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/go-gl/mathgl/mgl32"
	"log"
	"math"
	"os"
	"runtime"
	"sort"
	"strings"
)

const (
	windowWidth      = 800
	windowHeight     = 600
	MouseSensitivity = 0.9
	PolygonSize      = 7
)

var (
	xAngle         = float32(0)
	zAngle         = float32(3)
	cameraPosition = mgl32.Vec3{-1024, -512, -512}
	lightmapSize   = int32(512)

	windowHandler *WindowHandler
)

type RenderMap struct {
	MapTextures []MapTexture
	MapLightmap MapLightmap
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

type RenderTriangle struct {
	X  float32
	Y  float32
	Z  float32
	U  float32
	V  float32
	LU float32
	LV float32
}

func GetViewMatrix() mgl32.Mat4 {
	matrix := mgl32.Ident4()
	matrix = matrix.Mul4(mgl32.HomogRotate3DX(xAngle - mgl32.DegToRad(90)))
	matrix = matrix.Mul4(mgl32.HomogRotate3DZ(zAngle))
	matrix = matrix.Mul4(mgl32.Translate3D(cameraPosition.X(), cameraPosition.Y(), cameraPosition.Z()))
	return matrix
}

func checkInput() {
	// Move the camera around using WASD keys
	speed := float32(200 * windowHandler.getTimeSinceLastFrame())
	dir := []float32{0, 0, 0}
	if windowHandler.inputHandler.isActive(PLAYER_FORWARD) {
		dir[2] += speed
	} else if windowHandler.inputHandler.isActive(PLAYER_BACKWARD) {
		dir[2] -= speed
	} else if windowHandler.inputHandler.isActive(PLAYER_LEFT) {
		dir[0] += speed
	} else if windowHandler.inputHandler.isActive(PLAYER_RIGHT) {
		dir[0] -= speed
	}

	cameraMatrix := mgl32.Ident4()
	cameraMatrix = cameraMatrix.Mul4(mgl32.HomogRotate3DX(xAngle - mgl32.DegToRad(90)))
	cameraMatrix = cameraMatrix.Mul4(mgl32.HomogRotate3DZ(zAngle))
	cameraMatrix = cameraMatrix.Inv()
	movementDelta := cameraMatrix.Mul4x1(mgl32.Vec4{dir[0], dir[1], dir[2], 0.0})

	cameraPosition = cameraPosition.Add(mgl32.Vec3{movementDelta.X(), movementDelta.Y(), movementDelta.Z()})

	offset := windowHandler.inputHandler.getCursorChange()
	xOffset := float32(offset[0] * MouseSensitivity)
	yOffset := float32(offset[1] * MouseSensitivity)

	zAngle += xOffset * 0.025
	for zAngle < 0 {
		zAngle += math.Pi * 2
	}
	for zAngle >= math.Pi*2 {
		zAngle -= math.Pi * 2
	}

	xAngle += yOffset * 0.025
	for xAngle < -math.Pi*0.5 {
		xAngle = -math.Pi * 0.5
	}
	for xAngle > math.Pi*0.5 {
		xAngle = math.Pi * 0.5
	}
}

func initOpenGL() uint32 {
	if err := gl.Init(); err != nil {
		panic(err)
	}

	version := gl.GoStr(gl.GetString(gl.VERSION))
	fmt.Println("OpenGL version", version)

	shader := render.NewShader()
	return shader.ProgramShader
}

// Initialize texture in OpenGL using image data
func buildTexture(imageData []uint8, walData render.WalHeader) uint32 {
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
	stride := int32(PolygonSize * floatSize)

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

	// Bind the lightmap texture
	gl.ActiveTexture(gl.TEXTURE1)
	gl.BindTexture(gl.TEXTURE_2D, renderMap.MapLightmap.Texture)
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

func getEdgeVertex(mapData *render.MapData, faceEdgeIdx int) render.Vertex {
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

func getTextureUV(vtx render.Vertex, tex render.TexInfo) [2]float32 {
	u := float32(vtx.X*tex.UAxis[0] + vtx.Y*tex.UAxis[1] + vtx.Z*tex.UAxis[2] + tex.UOffset)
	v := float32(vtx.X*tex.VAxis[0] + vtx.Y*tex.VAxis[1] + vtx.Z*tex.VAxis[2] + tex.VOffset)
	return [2]float32{u, v}
}

func getTextureFilename(texInfo render.TexInfo) string {
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

func createTextureList(textureIds map[string]int) []MapTexture {
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
		fullFilename := "./data/textures/" + strings.Trim(fileKeys[i], " ") + ".wal"

		texFile, _ := os.Open(fullFilename)
		defer texFile.Close()

		if texFile == nil {
			log.Fatal("Texture file doesn't exist")
			return nil
		}

		imageData, walData, err := render.LoadQ2WAL(texFile)
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

func createTriangleData(mapData *render.MapData, mapTextures []MapTexture) ([]float32, RenderMap) {
	vertsByTexture := make(map[int][]RenderTriangle)

	lightmap := initializeLightmap()

	var offset uint16
	for faceIdx := 0; faceIdx < len(mapData.Faces); faceIdx++ {
		faceInfo := mapData.Faces[faceIdx]
		texInfo := mapData.TexInfos[faceInfo.TextureInfo]

		// hide skybox
		if texInfo.Flags&4 != 0 {
			continue
		}

		// get index in texture array
		filename := getTextureFilename(texInfo)
		texId := mapData.TextureIds[filename]

		_, ok := vertsByTexture[texId]
		if !ok {
			vertsByTexture[texId] = make([]RenderTriangle, 0)
		}

		v0 := getEdgeVertex(mapData, int(faceInfo.FirstEdge))
		uv0 := getTextureUV(v0, texInfo)
		v1 := getEdgeVertex(mapData, int(faceInfo.FirstEdge)+1)
		uv1 := getTextureUV(v1, texInfo)

		// Generate triangle fan from polyglon
		var faceData []RenderTriangle
		for offset = 2; offset < faceInfo.NumEdges; offset++ {
			v2 := getEdgeVertex(mapData, int(faceInfo.FirstEdge)+int(offset))
			uv2 := getTextureUV(v2, texInfo)

			// Add triangle
			faceData = append(faceData, RenderTriangle{v0.X, v0.Y, v0.Z, uv0[0], uv0[1], 0.999, 0.999})
			faceData = append(faceData, RenderTriangle{v1.X, v1.Y, v1.Z, uv1[0], uv1[1], 0.999, 0.999})
			faceData = append(faceData, RenderTriangle{v2.X, v2.Y, v2.Z, uv2[0], uv2[1], 0.999, 0.999})

			v1 = v2
			uv1 = uv2
		}

		if texInfo.Flags == 0 {
			processLightmap(&lightmap, faceData, texInfo, faceInfo.LightmapOffset, mapData)
		}

		// add all triangle data for this texture
		for j := 0; j < len(faceData); j++ {
			vertsByTexture[texId] = append(vertsByTexture[texId], faceData[j])
		}
	}

	// Generate mipmaps for the lightmap
	gl.BindTexture(gl.TEXTURE_2D, lightmap.Texture)
	gl.GenerateMipmap(gl.TEXTURE_2D)

	// only get the textures that were used in the map
	var texKeys []int
	for k, _ := range vertsByTexture {
		texKeys = append(texKeys, k)
	}
	sort.Ints(texKeys)

	// allocate a buffer
	bufferSize := 0
	for i := 0; i < len(texKeys); i++ {
		bufferSize += int(len(vertsByTexture[texKeys[i]])) * PolygonSize
	}

	// rearrange data by texture
	copyMapTextures := mapTextures[:]
	fullBuffer := make([]float32, bufferSize)
	bufferOffset := 0
	for i := 0; i < len(texKeys); i++ {
		texVertSize := int32(len(vertsByTexture[texKeys[i]]))
		copyMapTextures[texKeys[i]].VertOffset = int32(bufferOffset / PolygonSize)
		copyMapTextures[texKeys[i]].VertCount = texVertSize

		for j := 0; j < int(texVertSize); j++ {
			arr := vertsByTexture[texKeys[i]]
			x := arr[j].X
			y := arr[j].Y
			z := arr[j].Z

			textureWidth := float32(copyMapTextures[texKeys[i]].Width)
			textureHeight := float32(copyMapTextures[texKeys[i]].Height)
			u := arr[j].U / textureWidth
			v := arr[j].V / textureHeight

			lu := arr[j].LU
			lv := arr[j].LV

			// Position
			fullBuffer[bufferOffset+0] = x
			fullBuffer[bufferOffset+1] = y
			fullBuffer[bufferOffset+2] = z

			// Texture UV
			fullBuffer[bufferOffset+3] = u
			fullBuffer[bufferOffset+4] = v

			// Lightmap UV
			fullBuffer[bufferOffset+5] = lu
			fullBuffer[bufferOffset+6] = lv

			bufferOffset += PolygonSize
		}
	}

	renderMap := RenderMap{
		MapLightmap: lightmap,
		MapTextures: copyMapTextures,
	}
	return fullBuffer, renderMap
}

func initializeLightmap() MapLightmap {
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
	return MapLightmap{
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

func processLightmap(lightmap *MapLightmap, faceData []RenderTriangle, texInfo render.TexInfo, offset uint32, mapData *render.MapData) {
	size := getLightmapDimensions(faceData)

	rect := readLightmap(lightmap, offset, size.Width, size.Height, mapData)
	if rect != nil {
		// Determine the correct TexCoords for the lightmap
		for i := 0; i < len(faceData); i++ {
			x := faceData[i].X
			y := faceData[i].Y
			z := faceData[i].Z

			s := ((x*texInfo.UAxis[0] + y*texInfo.UAxis[1] + z*texInfo.UAxis[2]) + texInfo.UOffset) - size.MinU
			s += float32((rect.X * 16) + 8)
			s /= float32(lightmapSize * 16)

			t := ((x*texInfo.VAxis[0] + y*texInfo.VAxis[1] + z*texInfo.VAxis[2]) + texInfo.VOffset) - size.MinV
			t += float32((rect.Y * 16) + 8)
			t /= float32(lightmapSize * 16)

			faceData[i].LU = s
			faceData[i].LV = t
		}
	}
}

// Get the width and height of the lightmap
func getLightmapDimensions(faceData []RenderTriangle) LightmapDimensions {
	// Find the Min and Max UV's for a face
	startUV0 := float64(faceData[0].U)
	startUV1 := float64(faceData[0].V)
	minU := math.Floor(startUV0)
	minV := math.Floor(startUV1)
	maxU := math.Floor(startUV0)
	maxV := math.Floor(startUV1)

	for i := 1; i < len(faceData); i++ {
		uv0 := float64(faceData[i].U)
		uv1 := float64(faceData[i].V)

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

func readLightmap(lightmap *MapLightmap, offset uint32, width int32, height int32, mapData *render.MapData) *LightmapNode {
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
			r := mapData.LightmapData[baseIndex+0]
			g := mapData.LightmapData[baseIndex+1]
			b := mapData.LightmapData[baseIndex+2]

			bytes[curByte+0] = r
			bytes[curByte+1] = g
			bytes[curByte+2] = b
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

func initMesh(bspFilename string) (*render.MapData, []MapTexture, error) {
	file, err := os.Open(bspFilename)
	defer file.Close()

	if file == nil {
		log.Fatal("BSP file doesn't exist")
		return nil, nil, err
	}

	mapData, err := render.LoadQ2BSP(file)
	if err != nil {
		log.Fatal("Error loading bsp in main:", err)
		return nil, nil, err
	}
	fmt.Println("Map data successfully loaded")

	oldMapTextures := createTextureList(mapData.TextureIds)
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
	mapData, oldMapTextures, err := initMesh("./data/test.bsp")
	if err != nil {
		fmt.Println("Error initializing mesh: ", err)
		return
	}

	triangleData, renderMap := createTriangleData(mapData, oldMapTextures)
	fmt.Println("Rendering data is generated. Begin rendering.")

	for !windowHandler.shouldClose() {
		windowHandler.startFrame()

		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		// Activate shader
		gl.UseProgram(programShader)

		// Create transformations
		view := GetViewMatrix()
		ratio := float64(windowWidth) / float64(windowHeight)
		projection := mgl32.Perspective(45.0, float32(ratio), 0.1, 4096.0)

		// Get their uniform location
		viewLoc := gl.GetUniformLocation(programShader, gl.Str("view\x00"))
		projectionLoc := gl.GetUniformLocation(programShader, gl.Str("projection\x00"))

		// Pass the matrices to the shader
		gl.UniformMatrix4fv(viewLoc, 1, false, &view[0])
		gl.UniformMatrix4fv(projectionLoc, 1, false, &projection[0])

		// Render map data to the screen
		drawMap(triangleData, renderMap, programShader)

		checkInput()
	}
}
