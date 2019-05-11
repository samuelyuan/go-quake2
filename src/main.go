package main

import (
	"./render"
	"fmt"
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/go-gl/mathgl/mgl64"
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
	MouseSensitivity = 0.1
)

var (
	position    = mgl32.Vec3{0, 0, 3}.Normalize()
	cameraFront = mgl32.Vec3{0, 0, -1}.Normalize()
	cameraUp    = mgl32.Vec3{0, 1, 3}.Normalize()
	right       = cameraFront.Cross(cameraUp).Normalize()

	// Used for movement
	deltaTime = 0.0
	lastFrame = 0.0

	// Eular angles (in degrees)
	yaw   = -90.0
	pitch = 0.0

	// Mouse settings
	firstMouse = true
	lastX      float64
	lastY      float64

	pressed 	[256]bool
)

type MapTexture struct {
	Id uint32
	Width uint32
	Height uint32
	VertOffset int32
	VertCount  int32
}

// Resize the screen
func resizeCallback(w *glfw.Window, width int, height int) {
	gl.Viewport(0, 0, int32(width), int32(height))
}

func mouseCallback(w *glfw.Window, xPos float64, yPos float64) {
	if firstMouse {
		lastX = xPos
		lastY = yPos
		firstMouse = false
	}

	xOffset := xPos - lastX
	// reversed
	yOffset := lastY - yPos

	lastX = xPos
	lastY = yPos

	xOffset *= MouseSensitivity
	yOffset *= MouseSensitivity

	yaw += xOffset
	pitch += yOffset

	// Make sure that when pitch is out of bounds, screen doesn't get flipped
	if pitch > 89.0 {
		pitch = 89.0
	}
	if pitch < -89.0 {
		pitch = -89.0
	}

	// Update vectors using the updated Euler angles
	x := float32(math.Cos(mgl64.DegToRad(yaw)) * math.Cos(mgl64.DegToRad(pitch)))
	y := float32(math.Sin(mgl64.DegToRad(pitch)))
	z := float32(math.Sin(mgl64.DegToRad(yaw)) * math.Cos(mgl64.DegToRad(pitch)))
	front := mgl32.Vec3{x, y, z}

	// recalculate vectors
	cameraFront = front.Normalize()
	right = cameraFront.Cross(cameraUp).Normalize()
}

func keyCallback(window *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	if action == glfw.Release {
		pressed[glfw.KeyW] = false
		pressed[glfw.KeyS] = false
		pressed[glfw.KeyA] = false
		pressed[glfw.KeyD] = false
	}

	if action == glfw.Press {
		// Quit the program if the escape key is pressed
		if key == glfw.KeyEscape {
			window.SetShouldClose(true)
		}

		if key == glfw.KeyW {
			pressed[glfw.KeyW] = true
		}
		if key == glfw.KeyS {
			pressed[glfw.KeyS] = true
		}
		if key == glfw.KeyA {
			pressed[glfw.KeyA] = true
		}
		if key == glfw.KeyD {
			pressed[glfw.KeyD] = true
		}
	}

	// Move the camera around using WASD keys
	// Set frame time
	currentFrame := glfw.GetTime()
	deltaTime = currentFrame - lastFrame
	lastFrame = currentFrame

	velocity := float32(0.5 * deltaTime)

	if (pressed[glfw.KeyW]) {
		// forward
		position = position.Add(cameraFront.Mul(velocity))
	} else if (pressed[glfw.KeyS]) {
		// backward
		position = position.Sub(cameraFront.Mul(velocity))
	} else if (pressed[glfw.KeyA]) {
		// left
		position = position.Sub(right.Mul(velocity))
	} else if (pressed[glfw.KeyD]) {
		// right
		position = position.Add(right.Mul(velocity))
	}
}

func GetViewMatrix() mgl32.Mat4 {
	eye := position
	center := position.Add(cameraFront)
	return mgl32.LookAt(
		eye.X(), eye.Y(), eye.Z(),
		center.X(), center.Y(), center.Z(),
		cameraUp.X(), cameraUp.Y(), cameraUp.Z())
}

func initGLFW() *glfw.Window {
	if err := glfw.Init(); err != nil {
		panic(fmt.Errorf("Could not initialize glfw: %v", err))
	}

	// Initialize and create window
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.Resizable, glfw.True)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

	window, err := glfw.CreateWindow(windowWidth, windowHeight, "Quake 2 BSP Loader", nil, nil)
	if err != nil {
		panic(fmt.Errorf("Could not create OpenGL renderer: %v", err))
	}
	window.MakeContextCurrent()

	// Check for resize
	window.SetSizeCallback(resizeCallback)
	window.GetSize()

	// Keyboard callback
	window.SetKeyCallback(keyCallback)
	// Mouse callback
	window.SetCursorPosCallback(mouseCallback)

	return window
}

func initOpenGL() uint32 {
	if err := gl.Init(); err != nil {
		panic(err)
	}

	lastX = windowWidth / 2.0
	lastY = windowHeight / 2.0
	firstMouse = true

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

		if (texture.VertCount == 0) {
			continue;
		}

		// Bind the texture
		gl.ActiveTexture(gl.TEXTURE0);
		gl.BindTexture(gl.TEXTURE_2D, texture.Id);

		// Draw all faces for this texture
		gl.DrawArrays(gl.TRIANGLES, texture.VertOffset, texture.VertCount)
	}

	return
}

func getVertex(mapData *render.MapData, faceEdgeIdx int) render.Vertex {
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

func createTriangleData(mapData *render.MapData, mapTextures []MapTexture) ([]float32, []MapTexture) {
	vertsByTexture := make(map[int][]float32)

	var offset uint16
	for faceIdx := 0; faceIdx < len(mapData.Faces); faceIdx++ {
		faceInfo := mapData.Faces[faceIdx]
		texInfo := mapData.TexInfos[faceInfo.TextureInfo]

		// hide skybox
		if (texInfo.Flags & 4 != 0) {
			continue;
		}

		// get index in texture array
		filename := getTextureFilename(texInfo)
		texId := mapData.TextureIds[filename]

		_, ok := vertsByTexture[texId]
		if !ok {
			vertsByTexture[texId] = make([]float32, 0)
		}

		v0 := getVertex(mapData, int(faceInfo.FirstEdge))
		uv0 := getTextureUV(v0, texInfo)
		v1 := getVertex(mapData, int(faceInfo.FirstEdge)+1)
		uv1 := getTextureUV(v1, texInfo)

		// Generate triangle fan from polyglon
		var faceData []float32
		for offset = 2; offset < faceInfo.NumEdges; offset++ {
			v2 := getVertex(mapData, int(faceInfo.FirstEdge)+int(offset))
			uv2 := getTextureUV(v2, texInfo)

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
			x := arr[j + 0]
			y := arr[j + 1]
			z := arr[j + 2]

			u := arr[j + 3]
			v := arr[j + 4]

			// Position
			scale := float32(500.0)
			fullBuffer[bufferOffset + 0] = x / scale
			fullBuffer[bufferOffset + 1] = y / scale
			fullBuffer[bufferOffset + 2] = z / scale

			// UV
			width := float32(copyMapTextures[texKeys[i]].Width)
			height := float32(copyMapTextures[texKeys[i]].Height)
			fullBuffer[bufferOffset + 3] = u / width
			fullBuffer[bufferOffset + 4] = v / height

			bufferOffset += 5
		}
	}

	return fullBuffer, copyMapTextures
}

func main() {
	// Load files
	fmt.Println("Starting quake2 bsp loader\n")

	file, _ := os.Open("./data/test.bsp")
	defer file.Close()

	if file == nil {
		log.Fatal("BSP file doesn't exist")
		return
	}

	mapData, err := render.LoadQ2BSP(file)
	if err != nil {
		log.Fatal("Error loading bsp in main:", err)
		return
	}
	fmt.Println("Map data successfully loaded")

	// Run OpenGL code
	runtime.LockOSThread()
	window := initGLFW()
	programShader := initOpenGL()

	gl.ClearColor(0.0, 0.0, 0.0, 1.0)

	// get sorted strings
	var fileKeys []string
	for texFilename, _ := range mapData.TextureIds {
		fileKeys = append(fileKeys, texFilename)
	}
	sort.Strings(fileKeys)

	// iterate through filenames in the same order
	oldMapTextures := make([]MapTexture, len(fileKeys))
	for i := 0; i < len(fileKeys); i++{
		// stored in different folder
		// append extension (.wal) as default
		fullFilename := "./data/textures/" + strings.Trim(fileKeys[i], " ") + ".wal"

		texFile, _ := os.Open(fullFilename)
		defer texFile.Close()

		if texFile == nil {
			log.Fatal("Texture file doesn't exist")
			return
		}

		imageData, walData, err := render.LoadQ2WAL(texFile)
		if err != nil {
			log.Fatal("Error loading texture in main:", err)
			return
		}

		// Initialize texture
		texId := buildTexture(imageData, walData)

		// the index is not necessarily in order
		index := mapData.TextureIds[fileKeys[i]]
		oldMapTextures[index] = MapTexture{}
		oldMapTextures[index].Width = walData.Width
		oldMapTextures[index].Height = walData.Height
		// opengl texture id
		oldMapTextures[index].Id = texId
	}
	fmt.Println("Textures successfully loaded")

	triangleData, mapTextures := createTriangleData(mapData, oldMapTextures)

	for !window.ShouldClose() {
		gl.Enable(gl.DEPTH_TEST)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		// Activate shader
		gl.UseProgram(programShader)

		// Create transformations
		view := GetViewMatrix()
		ratio := float64(windowWidth) / float64(windowHeight)
		projection := mgl32.Perspective(45.0, float32(ratio), 0.1, 100.0)

		// Get their uniform location
		viewLoc := gl.GetUniformLocation(programShader, gl.Str("view\x00"))
		projectionLoc := gl.GetUniformLocation(programShader, gl.Str("projection\x00"))

		// Pass the matrices to the shader
		gl.UniformMatrix4fv(viewLoc, 1, false, &view[0])
		gl.UniformMatrix4fv(projectionLoc, 1, false, &projection[0])

		// Render map data to the screen
		drawMap(triangleData, mapTextures, programShader)

		// Window events for keyboard and mouse
		glfw.PollEvents()
		window.SwapBuffers()
	}
}
