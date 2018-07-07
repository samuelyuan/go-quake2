package main

import (
	"fmt"
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/go-gl/mathgl/mgl64"
	"log"
	"math"
	"os"
	"runtime"
	"strings"
)

const (
	windowWidth      = 800
	windowHeight     = 600
	MouseSensitivity = 0.05

	vertexShaderSource = `
		#version 410
	  layout (location = 0) in vec3 position;

    uniform mat4 view;

		void main() {
			gl_Position = view * vec4(position, 1.0);
		}
	` + "\x00"

	fragmentShaderSource = `
		#version 410
		out vec4 frag_colour;
		void main() {
			frag_colour = vec4(1, 1, 1, 1.0);
		}
	` + "\x00"
)

var (
	position = mgl32.Vec3{0, 0, 3}.Normalize()
	forward  = mgl32.Vec3{0, 0, -1}.Normalize()
	up       = mgl32.Vec3{0, 1, 3}.Normalize()
	right    = forward.Cross(up).Normalize()

	// Used for movement
	deltaTime float64
	lastFrame float64

	// Eular angles (in degrees)
	yaw   = -90.0
	pitch = 0.0

	// Mouse settings
	firstMouse = true
	lastX      float64
	lastY      float64
)

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
	forward = front.Normalize()

	// recalculate vectors
	right = front.Cross(up).Normalize()
}

func keyCallback(window *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	if action == glfw.Press {
		// Quit the program if the escape key is pressed
		if key == glfw.KeyEscape {
			window.SetShouldClose(true)
		}

		// Move the camera around using WASD keys
		// Set frame time
		currentFrame := glfw.GetTime()
		deltaTime = currentFrame - lastFrame
		lastFrame = currentFrame

		velocity := float32(0.05 * deltaTime)
		if key == glfw.KeyW {
			// forward
			position = position.Add(forward.Mul(velocity))
		}
		if key == glfw.KeyS {
			// backward
			position = position.Sub(forward.Mul(velocity))
		}
		if key == glfw.KeyA {
			// left
			position = position.Sub(right.Mul(velocity))
		}
		if key == glfw.KeyD {
			// right
			position = position.Add(right.Mul(velocity))
		}
	}
}

func GetViewMatrix() mgl32.Mat4 {
	eye := position
	center := position.Add(forward)
	return mgl32.LookAt(
		eye.X(), eye.Y(), eye.Z(),
		center.X(), center.Y(), center.Z(),
		up.X(), up.Y(), up.Z())
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

func compileShader(source string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)

	csources, free := gl.Strs(source)
	gl.ShaderSource(shader, 1, csources, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("Failed to compile %v: %v", source, log)
	}

	return shader, nil
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

	// compile shaders
	vertexShader, err := compileShader(vertexShaderSource, gl.VERTEX_SHADER)
	if err != nil {
		panic(err)
	}
	fragmentShader, err := compileShader(fragmentShaderSource, gl.FRAGMENT_SHADER)
	if err != nil {
		panic(err)
	}

	programShader := gl.CreateProgram()
	gl.AttachShader(programShader, vertexShader)
	gl.AttachShader(programShader, fragmentShader)
	gl.LinkProgram(programShader)

	return programShader
}

func makeVertexArrayObj(vertices []float32) uint32 {
	var vao uint32
	var vbo uint32

	// Create buffers/arrays
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)

	// Load data
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	structSize := 4 // size of float32 is 4
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*structSize, gl.Ptr(vertices), gl.STATIC_DRAW)

	// Position attribute
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 0, nil)
	gl.EnableVertexAttribArray(0)

	return vao
}

func createVertexData(mapData *MapData) []float32 {
	var vertexData []float32

	for edgeIdx := 0; edgeIdx < len(mapData.Edges); edgeIdx++ {
		v1Idx := mapData.Edges[edgeIdx].V1
		v2Idx := mapData.Edges[edgeIdx].V2

		v1Data := mapData.Vertices[v1Idx]
		v2Data := mapData.Vertices[v2Idx]

		// scale down to make the map fit the screen
		vertexData = append(vertexData, v1Data.X/500.0, v1Data.Y/500.0, v1Data.Z/500.0)
		vertexData = append(vertexData, v2Data.X/500.0, v2Data.Y/500.0, v2Data.Z/500.0)
	}

	return vertexData
}

func main() {
	// Load file
	fmt.Println("Starting quake2 bsp loader\n")

	file, _ := os.Open("test.bsp")
	defer file.Close()

	mapData, err := loadQ2BSP(file)
	if err != nil {
		log.Fatal(err)
		return
	}

	fmt.Println("File successfully loaded")

	// Run OpenGL code
	runtime.LockOSThread()
	window := initGLFW()
	programShader := initOpenGL()

	//gl.ClearColor(0, 0.5, 1.0, 1.0)

	vertexData := createVertexData(mapData)
	vertexArrayObj := makeVertexArrayObj(vertexData)

	for !window.ShouldClose() {
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		// Activate shader
		gl.UseProgram(programShader)

		// Create transformations
		view := GetViewMatrix()
		// Get their uniform location
		viewLoc := gl.GetUniformLocation(programShader, gl.Str("view\x00"))
		// Pass the matrices to the shader
		gl.UniformMatrix4fv(viewLoc, 1, false, &view[0])

		// Render map data to the screen
		gl.BindVertexArray(vertexArrayObj)
		gl.DrawArrays(gl.LINES, 0, int32(len(vertexData))/2)

		glfw.PollEvents()
		window.SwapBuffers()
	}
}
