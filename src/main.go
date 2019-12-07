package main

import (
	"./render"
	"fmt"
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/go-gl/glfw/v3.2/glfw"
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
)

var (
	xAngle         = float32(0)
	zAngle         = float32(3)
	cameraPosition = mgl32.Vec3{-1024, -512, -512}

	windowHandler *WindowHandler
)

type MapTexture struct {
	Id         uint32
	Width      uint32
	Height     uint32
	VertOffset int32
	VertCount  int32
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
	// velocity := float32(0.5 * windowHandler.getTimeSinceLastFrame())

	speed := float32(15)
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

func drawMap(vertices []float32, mapTextures []MapTexture, programShader uint32) {
	var vao uint32
	var vbo uint32

	// Create buffers/arrays
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)

	// Load data
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	floatSize := 4 // size of float32 is 4
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*floatSize, gl.Ptr(vertices), gl.STATIC_DRAW)

	// 3 floats for vertex, 2 floats for texture UV
	stride := int32(5 * floatSize)

	// Position attribute
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, stride, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)

	// Texture
	gl.VertexAttribPointer(1, 2, gl.FLOAT, false, stride, gl.PtrOffset(3*floatSize))
	gl.EnableVertexAttribArray(1)

	diffuseUniform := gl.GetUniformLocation(programShader, gl.Str("diffuse\x00"))
	gl.Uniform1i(diffuseUniform, 0)

	// Since faces are sorted by texture, we loop through all textures in the map
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

func getTextureUV(vtx render.Vertex, tex render.TexInfo, mapTexture MapTexture) [2]float32 {
	u := float32(vtx.X*tex.UAxis[0] + vtx.Y*tex.UAxis[1] + vtx.Z*tex.UAxis[2] + tex.UOffset)
	v := float32(vtx.X*tex.VAxis[0] + vtx.Y*tex.VAxis[1] + vtx.Z*tex.VAxis[2] + tex.VOffset)
	return [2]float32{u / float32(mapTexture.Width), v / float32(mapTexture.Height)}
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

func createTriangleData(mapData *render.MapData, mapTextures []MapTexture) ([]float32, []MapTexture) {
	vertsByTexture := make(map[int][]float32)

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
			vertsByTexture[texId] = make([]float32, 0)
		}

		mapTexture := mapTextures[texId]

		v0 := getEdgeVertex(mapData, int(faceInfo.FirstEdge))
		uv0 := getTextureUV(v0, texInfo, mapTexture)
		v1 := getEdgeVertex(mapData, int(faceInfo.FirstEdge)+1)
		uv1 := getTextureUV(v1, texInfo, mapTexture)

		// Generate triangle fan from polyglon
		var faceData []float32
		for offset = 2; offset < faceInfo.NumEdges; offset++ {
			v2 := getEdgeVertex(mapData, int(faceInfo.FirstEdge)+int(offset))
			uv2 := getTextureUV(v2, texInfo, mapTexture)

			// Add triangle
			faceData = append(faceData, v0.X, v0.Y, v0.Z, uv0[0], uv0[1])
			faceData = append(faceData, v1.X, v1.Y, v1.Z, uv1[0], uv1[1])
			faceData = append(faceData, v2.X, v2.Y, v2.Z, uv2[0], uv2[1])

			v1 = v2
			uv1 = uv2
		}

		// add all triangle data for this texture
		for j := 0; j < len(faceData); j++ {
			vertsByTexture[texId] = append(vertsByTexture[texId], faceData[j])
		}
	}

	// only get the textures that were used in the map
	var texKeys []int
	for k, _ := range vertsByTexture {
		texKeys = append(texKeys, k)
	}
	sort.Ints(texKeys)

	// allocate a buffer
	bufferSize := 0
	for i := 0; i < len(texKeys); i++ {
		bufferSize += int(len(vertsByTexture[texKeys[i]]))
	}

	// rearrange data by texture
	copyMapTextures := mapTextures[:]
	fullBuffer := make([]float32, bufferSize)
	bufferOffset := 0
	for i := 0; i < len(texKeys); i++ {
		texVertSize := int32(len(vertsByTexture[texKeys[i]]))
		copyMapTextures[texKeys[i]].VertOffset = int32(bufferOffset / 5)
		copyMapTextures[texKeys[i]].VertCount = int32(texVertSize / 5)

		for j := 0; j < int(texVertSize); j += 5 {
			arr := vertsByTexture[texKeys[i]]
			x := arr[j+0]
			y := arr[j+1]
			z := arr[j+2]

			u := arr[j+3]
			v := arr[j+4]

			// Position
			fullBuffer[bufferOffset+0] = x
			fullBuffer[bufferOffset+1] = y
			fullBuffer[bufferOffset+2] = z

			// UV
			fullBuffer[bufferOffset+3] = u
			fullBuffer[bufferOffset+4] = v

			bufferOffset += 5
		}
	}

	return fullBuffer, copyMapTextures
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
		fmt.Println("old map textures: ", oldMapTextures)
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

	// Load files
	mapData, oldMapTextures, err := initMesh("./data/test.bsp")
	if err != nil {
		fmt.Println("Error initializing mesh: ", err)
		return
	}

	triangleData, mapTextures := createTriangleData(mapData, oldMapTextures)

	for !windowHandler.shouldClose() {
		windowHandler.startFrame()

		gl.Enable(gl.DEPTH_TEST)
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
		drawMap(triangleData, mapTextures, programShader)

		checkInput()
	}
}
