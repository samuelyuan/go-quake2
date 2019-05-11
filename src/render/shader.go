package render

import (
	"fmt"
	"github.com/go-gl/gl/v4.1-core/gl"
	"strings"
)

const (
	vertexShaderSource = `
		#version 410
	  layout (location = 0) in vec3 position;
		layout (location = 1) in vec2 vertTexCoord;
		out vec2 fragTexCoord;

    uniform mat4 view;
		uniform mat4 projection;

		void main() {
			fragTexCoord = vertTexCoord;

			gl_Position = projection * view * vec4(position, 1.0);
		}
	` + "\x00"

	fragmentShaderSource = `
		#version 410

		uniform sampler2D diffuse;
		in vec2 fragTexCoord;
		out vec4 fragColor;

		void main() {
			vec4 diffuseColor = texture2D(diffuse, fragTexCoord.st);
			fragColor = diffuseColor;
		}
	` + "\x00"
)

type Shader struct {
	VertexShader   uint32
	FragmentShader uint32
	ProgramShader  uint32
}

func NewShader() *Shader {
	sh := Shader{}

	// compile shaders
	sh.VertexShader = sh.initVertexShader()
	sh.FragmentShader = sh.initFragmentShader()

	programShader := gl.CreateProgram()
	gl.AttachShader(programShader, sh.VertexShader)
	gl.AttachShader(programShader, sh.FragmentShader)
	gl.LinkProgram(programShader)
	sh.ProgramShader = programShader

	return &sh
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

func (sh *Shader) initVertexShader() uint32 {
	vertexShader, err := sh.compileShader(vertexShaderSource, gl.VERTEX_SHADER)
	if err != nil {
		panic(err)
	}
	return vertexShader
}

func (sh *Shader) initFragmentShader() uint32 {
	fragmentShader, err := sh.compileShader(fragmentShaderSource, gl.FRAGMENT_SHADER)
	if err != nil {
		panic(err)
	}
	return fragmentShader
}
