package main

import (
	"./q2file"
	"sort"
)

const (
	clusterInvalidId = ClusterId(65535)
)

type ClusterId uint16

type TreeLeaf struct {
	LeafIndex int   // index in bsp leaf array
	Faces     []int // contains face index in face array
}

type BSPTree struct {
	TreeLeaves []TreeLeaf
}

func NewBSPTree(mapData *q2file.MapData) *BSPTree {
	allFaceIds := make([]int, len(mapData.Faces))
	for faceIdx := 0; faceIdx < len(mapData.Faces); faceIdx++ {
		allFaceIds[faceIdx] = faceIdx
	}
	allLeaves, leavesInCluster := getLeavesInCluster(mapData)
	facesInCluster := getFacesInCluster(leavesInCluster)
	facesFromCluster := getFacesFromCluster(mapData, facesInCluster)
	// Use the PVS to get the full visibility data
	treeLeaves := getTreeLeaves(mapData, allLeaves, facesFromCluster, allFaceIds)
	return &BSPTree{
		TreeLeaves: treeLeaves,
	}
}

func getLeavesInCluster(mapData *q2file.MapData) ([]TreeLeaf, map[ClusterId][]TreeLeaf) {
	bspLeaves := mapData.BSPLeaves
	leafFacesTable := mapData.LeafFaces

	leavesInCluster := make(map[ClusterId][]TreeLeaf)
	allLeaves := make([]TreeLeaf, len(bspLeaves))
	for index, leaf := range bspLeaves {
		first := int(leaf.FirstLeafFace)

		faces := make([]int, int(leaf.NumLeafFaces))
		for offset := 0; offset < int(leaf.NumLeafFaces); offset++ {
			faces[offset] = int(leafFacesTable[first+offset])
		}

		c := ClusterId(leaf.Cluster)
		_, exists := leavesInCluster[c]
		if !exists {
			leavesInCluster[c] = make([]TreeLeaf, 0)
		}
		treeLeaf := TreeLeaf{
			LeafIndex: index,
			Faces:     faces,
		}
		allLeaves[index] = treeLeaf
		leavesInCluster[c] = append(leavesInCluster[c], treeLeaf)
	}

	return allLeaves, leavesInCluster
}

// Flatten the leaf faces into a single list
func getFacesInCluster(leavesInCluster map[ClusterId][]TreeLeaf) map[ClusterId][]int {
	facesInCluster := make(map[ClusterId][]int)
	for cluster, leaves := range leavesInCluster {
		visibleFaces := make([]int, 0)
		for _, leaf := range leaves {
			for _, faceId := range leaf.Faces {
				visibleFaces = append(visibleFaces, faceId)
			}
		}

		uniqueFaces := make(map[int]bool)
		for _, faceId := range visibleFaces {
			_, exists := uniqueFaces[faceId]
			if !exists {
				uniqueFaces[faceId] = true
			}
		}

		facesInCluster[cluster] = make([]int, 0)
		for faceId, _ := range uniqueFaces {
			facesInCluster[cluster] = append(facesInCluster[cluster], faceId)
		}
	}
	return facesInCluster
}

// Use PVS to calculate faces in other clusters that are visible from this cluster
func getFacesFromCluster(mapData *q2file.MapData, facesInCluster map[ClusterId][]int) map[ClusterId][]int {
	facesFromCluster := make(map[ClusterId][]int)
	for cluster, faces := range facesInCluster {
		visibleFaces := make([]int, 0)

		// copy existing faces
		for _, faceId := range faces {
			visibleFaces = append(visibleFaces, faceId)
		}

		if cluster == clusterInvalidId {
			continue
		}

		// PVS buffer index
		v := mapData.VisibilityOffsets[cluster].Pvs
		otherClusterIndex := 0
		numClusters := len(mapData.VisibilityOffsets)
		// Decompress the PVS
		for otherClusterIndex < numClusters {
			if mapData.VisibilityData[v] == 0 {
				// Zeros are run-length encoded. It encodes the number of zeros that should be there
				// to help compress the PVS, since most of it is empty
				v += 1
				otherClusterIndex += 8 * int(mapData.VisibilityData[v])
			} else {
				// Each entry in visibility data is a byte (8 bits)
				for bit := 0; bit < 8; bit++ {
					_, clusterExists := facesInCluster[ClusterId(otherClusterIndex)]
					if mapData.VisibilityData[v]&(1<<uint32(bit)) != 0 && clusterExists {
						for _, faceId := range facesInCluster[ClusterId(otherClusterIndex)] {
							visibleFaces = append(visibleFaces, faceId)
						}
					}
					otherClusterIndex += 1
				}
			}
			v += 1
		}

		uniqueFaces := make(map[int]bool)
		for _, faceId := range visibleFaces {
			_, exists := uniqueFaces[faceId]
			if !exists {
				uniqueFaces[faceId] = true
			}
		}

		facesFromCluster[cluster] = make([]int, 0)
		for faceId, _ := range uniqueFaces {
			facesFromCluster[cluster] = append(facesFromCluster[cluster], faceId)
		}
		sort.Ints(facesFromCluster[cluster])
	}
	return facesFromCluster
}

func getTreeLeaves(mapData *q2file.MapData, allLeaves []TreeLeaf, facesFromCluster map[ClusterId][]int, allFaceIds []int) []TreeLeaf {
	newLeafFaces := make([]TreeLeaf, len(allLeaves))
	bspLeaves := mapData.BSPLeaves
	for i, _ := range allLeaves {
		c := ClusterId(bspLeaves[i].Cluster)
		if c != clusterInvalidId {
			newLeafFaces[i] = TreeLeaf{
				LeafIndex: i,
				Faces:     facesFromCluster[c],
			}
		} else {
			newLeafFaces[i] = TreeLeaf{
				LeafIndex: i,
				Faces:     []int{},
			}
		}
	}

	return newLeafFaces
}

func (tree *BSPTree) findLeafNode(startNode int, mapData *q2file.MapData, position [3]float32) TreeLeaf {
	var d float32

	nodeId := startNode
	// Leaves have a negative node id
	for nodeId >= 0 {
		node := mapData.Nodes[int(nodeId)]
		plane := mapData.Planes[node.Plane]

		if plane.Type < uint32(3) {
			d = position[plane.Type] - plane.Distance
		} else {
			dotProduct := position[0]*plane.Normal[0] + position[1]*plane.Normal[1] + position[2]*plane.Normal[2]
			d = dotProduct - plane.Distance
		}

		if d < 0 {
			nodeId = int(node.BackChild)
		} else {
			nodeId = int(node.FrontChild)
		}
	}
	return tree.TreeLeaves[-(nodeId + 1)]
}
