#version 410
layout (location = 0) in vec3 position;
layout (location = 1) in vec2 vertTexCoord;
layout (location = 2) in vec2 texCoord2;
out vec2 fragTexCoord;
out vec2 vertexLightmapCoord;

uniform mat4 view;
uniform mat4 projection;

void main() {
  fragTexCoord = vertTexCoord;
  vertexLightmapCoord = texCoord2;

  gl_Position = projection * view * vec4(position, 1.0);
}
