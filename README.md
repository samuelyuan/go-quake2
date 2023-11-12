# go-quake2
Quake 2 Map Renderer written in Go and OpenGL.

<div style="display:inline-block;">
<img src="https://github.com/samuelyuan/go-quake2/raw/master/screenshots/map.png" alt="wireframe" width="400" height="300" />
</div>

### Features

* Loads any BSP file from Quake 2
* Free roam around the environment
* Renders only a small sector of the map depending on player location
* Supports static lightmapping

### Installation

1. Clone this project.
2. Get the game demo data. Download Quake 2 Demo and copy baseq2/pa0.pak from the Quake 2 directory to `data/` folder in this repository.
3. Run `go build`.

### Controls

- W/S to move forward/backward.
- A/D to move left/right.
- Use mouse to look around
