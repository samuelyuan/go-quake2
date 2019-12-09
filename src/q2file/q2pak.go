package q2file

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"unsafe"
)

type PakHeader struct {
	Magic  [4]byte // magic number ("PACK")
	Offset uint32
	Length uint32
}

type PakFile struct {
	Filename [56]byte
	Offset   uint32
	Length   uint32
}

func LoadQ2PAK(r io.ReaderAt) (map[string]PakFile, error) {
	pakHeader := PakHeader{}

	// Load header
	headerReader := io.NewSectionReader(r, 0, int64(unsafe.Sizeof(pakHeader)))
	if err := binary.Read(headerReader, binary.LittleEndian, &pakHeader); err != nil {
		return nil, err
	}

	// Verify format
	var magic = []byte("PACK")
	if !bytes.Equal(magic, pakHeader.Magic[:]) {
		return nil, fmt.Errorf("PAK Header: Wrong magic %v", pakHeader.Magic)
	}

	// Load file contents
	pakFileMap := make(map[string]PakFile)
	fileReader := io.NewSectionReader(r, int64(pakHeader.Offset), int64(pakHeader.Length))
	// Each PakFile is 64 bytes
	count := int(pakHeader.Length) / 64

	fmt.Println("PAK file contains ", count, " files")
	for i := 0; i < count; i++ {
		pakFile := PakFile{}
		if err := binary.Read(fileReader, binary.LittleEndian, &pakFile); err != nil {
			return nil, err
		}

		filename := byteToString(pakFile.Filename[:])
		pakFileMap[filename] = pakFile
	}
	return pakFileMap, nil
}

func LoadQ2BSPFromPAK(pakReader io.ReaderAt, pakFileMap map[string]PakFile, bspFilename string) (*MapData, error) {
	_, exists := pakFileMap[bspFilename]
	if !exists {
		return nil, fmt.Errorf("BSP filename %v doesn't exist in PAK", bspFilename)
	}

	bspReader := io.NewSectionReader(pakReader, int64(pakFileMap[bspFilename].Offset), int64(pakFileMap[bspFilename].Length))
	return LoadQ2BSP(bspReader)
}

func LoadQ2WALFromPAK(pakReader io.ReaderAt, pakFileMap map[string]PakFile, textureFilename string) ([]uint8, WalHeader, error) {
	_, exists := pakFileMap[textureFilename]
	if !exists {
		return nil, WalHeader{}, fmt.Errorf("Texture filename %v doesn't exist in PAK", textureFilename)
	}

	walReader := io.NewSectionReader(pakReader, int64(pakFileMap[textureFilename].Offset), int64(pakFileMap[textureFilename].Length))
	return LoadQ2WAL(walReader)
}

func byteToString(byteArr []byte) string {
	newString := ""
	for i := 0; i < len(byteArr); i++ {
		character := byteArr[i]
		// end of string
		if character == 0 {
			break
		}
		newString += string(character)
	}
	return newString
}
