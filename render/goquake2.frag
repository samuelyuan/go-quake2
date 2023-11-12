#version 410

uniform sampler2D diffuse;
uniform sampler2D lightmap;

in vec2 fragTexCoord;
in vec2 vertexLightmapCoord;
out vec4 fragColor;

void main() {
  vec4 diffuseColor = texture(diffuse, fragTexCoord.st);
  vec4 lightColor = texture(lightmap, vertexLightmapCoord.st);

  fragColor = vec4(diffuseColor.rgb * lightColor.rgb, diffuseColor.a);
}
