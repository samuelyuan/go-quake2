package render

type MapTexture struct {
	Id         uint32
	Width      uint32
	Height     uint32
	VertOffset int32
	VertCount  int32
}

func NewMapTexture(id uint32, width uint32, height uint32) MapTexture {
	texture := MapTexture{}
	texture.Id = id
	texture.Width = width
	texture.Height = height
	return texture
}
