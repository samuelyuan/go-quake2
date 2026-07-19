package q2file

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"unsafe"
)

const (
	LumpPlanes     = 1
	LumpVertices   = 2
	LumpVisibility = 3
	LumpBSPNodes   = 4
	LumpTexInfos   = 5
	LumpFaces      = 6
	LumpLightmaps  = 7
	LumpBSPLeaves  = 8
	LumpLeafFaces  = 9
	LumpEdges      = 11
	LumpFaceEdges  = 12
)

type Header struct {
	Magic   [4]byte  // magic number ("IBSP")
	Version uint32   // version of the BSP format (38)
	Lumps   [19]Lump // directory of the lumps
}

type Lump struct {
	Offset uint32 // offset (in bytes) of the data from the beginning of the file
	Length uint32 // length (in bytes) of the data
}

type Vertex struct {
	X float32
	Y float32
	Z float32
}

// Each edge is stored as a pair of indices into the vertex array
type Edge struct {
	V1 uint16
	V2 uint16
}

type Face struct {
	Plane     uint16 // index of the plane the face is parallel to
	PlaneSide uint16 // set if the normal is parallel to the plane normal

	FirstEdge uint32 // index of the first edge (in the face edge array)
	NumEdges  uint16 // number of consecutive edges (in the face edge array)

	TextureInfo uint16 // index of the texture info structure

	LightmapSyles  [4]uint8 // styles (bit flags) for the lightmaps
	LightmapOffset uint32   // offset of the lightmap (in bytes) in the lightmap lump
}

type FaceEdge struct {
	EdgeIndex int32
}

type TexInfo struct {
	UAxis       [3]float32
	UOffset     float32
	VAxis       [3]float32
	VOffset     float32
	Flags       uint32
	Value       uint32
	TextureName [32]byte
	NextTexInfo int32
}

type BSPNode struct {
	Plane uint32 // index of the splitting plane (in the plane array)

	FrontChild int32 // index of the front child node or leaf
	BackChild  int32 // index of the back child node or leaf

	BBoxMin [3]int16 // minimum x, y and z of the bounding box
	BBoxMax [3]int16 // maximum x, y and z of the bounding box

	FirstFace uint16 // index of the first face (in the face array)
	NumFaces  uint16 // number of consecutive edges (in the face array)
}

type Plane struct {
	Normal   [3]float32 // A, B, C components of the plane equation
	Distance float32    // D component of the plane equation
	Type     uint32
}

type BSPLeaf struct {
	BrushOr uint32

	Cluster uint16 // -1 for cluster indicates no visibility information
	Area    uint16

	BBoxMin [3]int16 // bounding box minimums
	BBoxMax [3]int16 // bounding box maximums

	FirstLeafFace uint16 // index of the first face (in the face leaf array)
	NumLeafFaces  uint16 // number of consecutive edges (in the face leaf array)

	FirstLeafBrush uint16
	NumLeafBrushes uint16
}

type LeafFace int16

type VisibilityOffset struct {
	Pvs uint32 // visibility set offset
	Phs uint32 // hearability set offset
}

type MapData struct {
	Vertices          []Vertex
	Edges             []Edge
	Faces             []Face
	FaceEdges         []FaceEdge
	TexInfos          []TexInfo
	TextureIds        map[string]int
	LightmapData      []uint8
	Nodes             []BSPNode
	Planes            []Plane
	BSPLeaves         []BSPLeaf
	LeafFaces         []LeafFace
	VisibilityData    []uint8
	VisibilityOffsets []VisibilityOffset
}

// Read header to verify the file is valid
// Parse the rest of the data and load it into a map
func LoadQ2BSP(r io.ReaderAt) (*MapData, error) {
	header := Header{}

	// Load header
	lumpReader := io.NewSectionReader(r, 0, int64(unsafe.Sizeof(header)))
	if err := binary.Read(lumpReader, binary.LittleEndian, &header); err != nil {
		return nil, err
	}

	// Verify format
	var magic = []byte("IBSP")
	if !bytes.Equal(magic, header.Magic[:]) {
		return nil, fmt.Errorf("BSP Header: Wrong magic %v", header.Magic)
	}

	if header.Version != 38 {
		return nil, fmt.Errorf("BSP Header: Wrong version %v", header.Version)
	}

	// Load map data
	fmt.Println("Header total lumps:", len(header.Lumps))

	vertices, err := loadLumpData[Vertex]("Vertex", header.Lumps[LumpVertices], r)
	if err != nil {
		return nil, fmt.Errorf("Failed to load vertices")
	}
	edges, err := loadLumpData[Edge]("Edge", header.Lumps[LumpEdges], r)
	if err != nil {
		return nil, fmt.Errorf("Failed to load edges")
	}
	faces, err := loadLumpData[Face]("Face", header.Lumps[LumpFaces], r)
	if err != nil {
		return nil, fmt.Errorf("Failed to load faces")
	}
	faceEdges, err := loadLumpData[FaceEdge]("Face edge", header.Lumps[LumpFaceEdges], r)
	if err != nil {
		return nil, fmt.Errorf("Failed to load face edges")
	}
	texInfos, err := loadLumpData[TexInfo]("Tex info", header.Lumps[LumpTexInfos], r)
	if err != nil {
		return nil, fmt.Errorf("Failed to load texture info")
	}

	textureIds := getTextureIds(texInfos)

	lightmapData, err := loadLumpData[uint8]("Lightmap data", header.Lumps[LumpLightmaps], r)
	if err != nil {
		return nil, fmt.Errorf("Failed to load lightmap data")
	}

	bspNodes, err := loadLumpData[BSPNode]("BSP Node", header.Lumps[LumpBSPNodes], r)
	if err != nil {
		return nil, fmt.Errorf("Failed to load BSP nodes")
	}
	planes, err := loadLumpData[Plane]("BSP Plane", header.Lumps[LumpPlanes], r)
	if err != nil {
		return nil, fmt.Errorf("Failed to load BSP planes")
	}
	bspLeaves, err := loadLumpData[BSPLeaf]("BSP Leaf", header.Lumps[LumpBSPLeaves], r)
	if err != nil {
		return nil, fmt.Errorf("Failed to load BSP leaves")
	}
	leafFaces, err := loadLumpData[LeafFace]("Leaf face", header.Lumps[LumpLeafFaces], r)
	if err != nil {
		return nil, fmt.Errorf("Failed to load leaf faces")
	}
	visibilityData, err := loadLumpData[uint8]("Visibility data", header.Lumps[LumpVisibility], r)
	if err != nil {
		return nil, fmt.Errorf("Failed to load visibility data")
	}
	visibilityOffsets, err := loadVisibilityOffsets(header.Lumps[LumpVisibility], r)
	if err != nil {
		return nil, fmt.Errorf("Failed to load visibility offsets")
	}

	// Combine into map data
	mapData := &MapData{
		Vertices:          vertices,
		Edges:             edges,
		Faces:             faces,
		FaceEdges:         faceEdges,
		TexInfos:          texInfos,
		TextureIds:        textureIds,
		LightmapData:      lightmapData,
		Nodes:             bspNodes,
		Planes:            planes,
		BSPLeaves:         bspLeaves,
		LeafFaces:         leafFaces,
		VisibilityData:    visibilityData,
		VisibilityOffsets: visibilityOffsets,
	}

	return mapData, nil
}

// loadLumpData reads a lump as a sequence of fixed-size elements of type T.
// All BSP lumps share this same layout (element count derived from lump byte length),
// so a single generic reader replaces the ten near-identical loaders this used to be.
func loadLumpData[T any](label string, lump Lump, r io.ReaderAt) ([]T, error) {
	var zero T
	elemSize := int(unsafe.Sizeof(zero))
	numItems := int(lump.Length) / elemSize

	fmt.Printf("%s count: %d\n", label, numItems)

	reader := io.NewSectionReader(r, int64(lump.Offset), int64(lump.Length))
	return readItems[T](reader, numItems)
}

// readItems reads count consecutive fixed-size elements of type T from reader.
func readItems[T any](reader io.Reader, count int) ([]T, error) {
	data := make([]T, count)
	for i := 0; i < count; i++ {
		if err := binary.Read(reader, binary.LittleEndian, &data[i]); err != nil {
			return nil, err
		}
	}
	return data, nil
}

func loadVisibilityOffsets(lump Lump, r io.ReaderAt) ([]VisibilityOffset, error) {
	reader := io.NewSectionReader(r, int64(lump.Offset), int64(lump.Length))

	// Read visibility cluster size at the beginning of the lump
	visibilityClusterSize := uint32(0)
	if err := binary.Read(reader, binary.LittleEndian, &visibilityClusterSize); err != nil {
		return nil, err
	}

	fmt.Println("Visibility offset cluster count:", visibilityClusterSize)

	// For every cluster, check the visibility state of other clusters
	return readItems[VisibilityOffset](reader, int(visibilityClusterSize))
}

// Map each texture name to an id
// There could be multiple textures with the same name.
func getTextureIds(texInfos []TexInfo) map[string]int {
	textureIds := make(map[string]int)
	nextId := 0
	for i := 0; i < len(texInfos); i++ {
		texInfo := texInfos[i]

		filename := BytesToString(texInfo.TextureName[:])

		// generate new id for texture if necessary
		_, exists := textureIds[filename]
		if !exists {
			textureIds[filename] = nextId
			nextId++
		}
	}
	return textureIds
}
