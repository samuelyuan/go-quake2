package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"unsafe"
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

type MapData struct {
	Vertices []Vertex
	Edges []Edge
}

// Read header to verify the file is valid
// Parse the rest of the data and load it into a map
func loadQ2BSP(r io.ReaderAt) (*MapData, error) {
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

	vertices, err := loadVertices(header.Lumps[2], r)
	if err != nil {
		return nil, fmt.Errorf("Failed to load vertices")
	}
	edges, err := loadEdges(header.Lumps[11], r)
	if err != nil {
		return nil, fmt.Errorf("Failed to load Edges")
	}

	// Combine into map data
	mapData := &MapData{
		Vertices: vertices,
		Edges: edges,
	}

	return mapData, nil
}

// Load all vertices
func loadVertices(lump Lump, r io.ReaderAt) ([]Vertex, error) {
	// Each vertex is 3 32-bit floats
	// 12 bytes per vertex
	numVerts := int(lump.Length / 12)

	fmt.Println("Vertex count: ", numVerts)

	var vertexData []Vertex

	// Read each vertex
	vertexReader := io.NewSectionReader(r, int64(lump.Offset), int64(lump.Length))
	for i := 0; i < numVerts; i++ {
		vertex := Vertex{}
		if err := binary.Read(vertexReader, binary.LittleEndian, &vertex); err != nil {
			return nil, err
		}

		// Add to array
		vertexData = append(vertexData, vertex)
	}

	return vertexData, nil
}

// Load all edges
func loadEdges(lump Lump, r io.ReaderAt) ([]Edge, error) {
	// Each edge is 2 unsigned shorts, one for each vertex
	// 4 bytes per edge
	numEdges := int(lump.Length / 4)

	fmt.Println("Edge count:", numEdges)

	var edgeData []Edge

	// Read each edge
	edgeReader := io.NewSectionReader(r, int64(lump.Offset), int64(lump.Length))
	for i := 0; i < numEdges; i++ {
		edge := Edge{}
		if err := binary.Read(edgeReader, binary.LittleEndian, &edge); err != nil {
			return nil, err
		}
		// Add to array
		edgeData = append(edgeData, edge)
	}

	return edgeData, nil
}
