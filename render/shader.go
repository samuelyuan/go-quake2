package render

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/go-gl/gl/v4.1-core/gl"
)

type Shader struct {
	VertexShader   uint32
	FragmentShader uint32
	ProgramShader  uint32
}

func NewShader(vertexFilePath string, fragmentFilePath string) (*Shader, error) {
	sh := Shader{}

	// compile shaders
	vertexShader, err := sh.initVertexShader(vertexFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to compile vertex shader: %w", err)
	}
	sh.VertexShader = vertexShader

	fragmentShader, err := sh.initFragmentShader(fragmentFilePath)
	if err != nil {
		// Clean up vertex shader if fragment shader fails
		gl.DeleteShader(vertexShader)
		return nil, fmt.Errorf("failed to compile fragment shader: %w", err)
	}
	sh.FragmentShader = fragmentShader

	programShader := gl.CreateProgram()
	gl.AttachShader(programShader, sh.VertexShader)
	gl.AttachShader(programShader, sh.FragmentShader)
	gl.LinkProgram(programShader)

	// Check if program linking succeeded
	var status int32
	gl.GetProgramiv(programShader, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(programShader, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(programShader, logLength, nil, gl.Str(log))

		// Clean up shaders
		gl.DeleteShader(vertexShader)
		gl.DeleteShader(fragmentShader)
		gl.DeleteProgram(programShader)

		return nil, fmt.Errorf("failed to link shader program: %v", log)
	}

	sh.ProgramShader = programShader
	return &sh, nil
}

func (sh *Shader) compileShader(source string, shaderType uint32) (uint32, error) {
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

func (sh *Shader) initVertexShader(filePath string) (uint32, error) {
	shaderCode, err := sh.readShaderCode(filePath)
	if err != nil {
		return 0, err
	}
	vertexShader, err := sh.compileShader(shaderCode, gl.VERTEX_SHADER)
	if err != nil {
		return 0, err
	}
	return vertexShader, nil
}

func (sh *Shader) initFragmentShader(filePath string) (uint32, error) {
	shaderCode, err := sh.readShaderCode(filePath)
	if err != nil {
		return 0, err
	}
	fragmentShader, err := sh.compileShader(shaderCode, gl.FRAGMENT_SHADER)
	if err != nil {
		return 0, err
	}
	return fragmentShader, nil
}

// Read shader code from file
func (sh *Shader) readShaderCode(filePath string) (string, error) {
	code := ""
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open shader file %s: %w", filePath, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		code += "\n" + scanner.Text()
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to read shader file %s: %w", filePath, err)
	}
	code += "\x00"
	return code, nil
}

// Cleanup frees OpenGL shader resources
func (sh *Shader) Cleanup() {
	if sh.ProgramShader != 0 {
		gl.DeleteProgram(sh.ProgramShader)
		sh.ProgramShader = 0
	}
	if sh.VertexShader != 0 {
		gl.DeleteShader(sh.VertexShader)
		sh.VertexShader = 0
	}
	if sh.FragmentShader != 0 {
		gl.DeleteShader(sh.FragmentShader)
		sh.FragmentShader = 0
	}
}
