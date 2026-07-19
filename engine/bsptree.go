package engine

import (
	"sort"

	"github.com/samuelyuan/go-quake2/q2file"
)

// ClusterID represents a BSP cluster identifier
type ClusterID uint16

// TreeLeaf represents a leaf node in the BSP tree
type TreeLeaf struct {
	LeafIndex int   // index in bsp leaf array
	Faces     []int // contains face index in face array
}

// BSPTree represents a Binary Space Partitioning tree for efficient rendering
type BSPTree struct {
	TreeLeaves []TreeLeaf
}

// NewBSPTree creates a new BSP tree from map data
func NewBSPTree(mapData *q2file.MapData) *BSPTree {
	// Initialize all face IDs
	allFaceIds := make([]int, len(mapData.Faces))
	for faceIdx := range mapData.Faces {
		allFaceIds[faceIdx] = faceIdx
	}

	// Build the tree structure
	allLeaves, leavesInCluster := getLeavesInCluster(mapData)
	facesInCluster := getFacesInCluster(leavesInCluster)
	facesFromCluster := getFacesFromCluster(mapData, facesInCluster)
	treeLeaves := getTreeLeaves(mapData, allLeaves, facesFromCluster, allFaceIds)

	return &BSPTree{
		TreeLeaves: treeLeaves,
	}
}

// getLeavesInCluster organizes leaves by their cluster IDs
func getLeavesInCluster(mapData *q2file.MapData) ([]TreeLeaf, map[ClusterID][]TreeLeaf) {
	bspLeaves := mapData.BSPLeaves
	leafFacesTable := mapData.LeafFaces

	leavesInCluster := make(map[ClusterID][]TreeLeaf)
	allLeaves := make([]TreeLeaf, len(bspLeaves))

	for index, leaf := range bspLeaves {
		first := int(leaf.FirstLeafFace)
		faces := make([]int, int(leaf.NumLeafFaces))

		for offset := 0; offset < int(leaf.NumLeafFaces); offset++ {
			faces[offset] = int(leafFacesTable[first+offset])
		}

		clusterID := ClusterID(leaf.Cluster)
		if _, exists := leavesInCluster[clusterID]; !exists {
			leavesInCluster[clusterID] = make([]TreeLeaf, 0)
		}

		treeLeaf := TreeLeaf{
			LeafIndex: index,
			Faces:     faces,
		}
		allLeaves[index] = treeLeaf
		leavesInCluster[clusterID] = append(leavesInCluster[clusterID], treeLeaf)
	}

	return allLeaves, leavesInCluster
}

// getFacesInCluster flattens leaf faces into cluster-based face lists
func getFacesInCluster(leavesInCluster map[ClusterID][]TreeLeaf) map[ClusterID][]int {
	facesInCluster := make(map[ClusterID][]int)

	for cluster, leaves := range leavesInCluster {
		visibleFaces := make([]int, 0)
		for _, leaf := range leaves {
			visibleFaces = append(visibleFaces, leaf.Faces...)
		}

		uniqueFaces := getUniqueFacesFromVisibleFaces(visibleFaces)
		facesInCluster[cluster] = getFaceIdsFromUniqueFaces(uniqueFaces)
	}

	return facesInCluster
}

// getFacesFromCluster uses PVS (Potentially Visible Set) to calculate visible faces
func getFacesFromCluster(mapData *q2file.MapData, facesInCluster map[ClusterID][]int) map[ClusterID][]int {
	facesFromCluster := make(map[ClusterID][]int)

	for cluster, faces := range facesInCluster {
		if cluster == ClusterID(ClusterInvalidID) {
			continue
		}

		// Start with existing faces
		visibleFaces := make([]int, len(faces))
		copy(visibleFaces, faces)

		// Process PVS data
		v := mapData.VisibilityOffsets[cluster].Pvs
		otherClusterIndex := 0
		numClusters := len(mapData.VisibilityOffsets)

		// Decompress the PVS
		for otherClusterIndex < numClusters {
			if mapData.VisibilityData[v] == 0 {
				// Zeros are run-length encoded
				v++
				otherClusterIndex += 8 * int(mapData.VisibilityData[v])
			} else {
				// Process each bit in the visibility byte
				for bit := 0; bit < 8; bit++ {
					if mapData.VisibilityData[v]&(1<<uint32(bit)) != 0 {
						if clusterFaces, exists := facesInCluster[ClusterID(otherClusterIndex)]; exists {
							visibleFaces = append(visibleFaces, clusterFaces...)
						}
					}
					otherClusterIndex++
				}
			}
			v++
		}

		// Remove duplicates and sort
		uniqueFaces := getUniqueFacesFromVisibleFaces(visibleFaces)
		facesFromCluster[cluster] = getFaceIdsFromUniqueFaces(uniqueFaces)
		sort.Ints(facesFromCluster[cluster])
	}

	return facesFromCluster
}

// getUniqueFacesFromVisibleFaces creates a set of unique face IDs
func getUniqueFacesFromVisibleFaces(visibleFaces []int) map[int]bool {
	uniqueFaces := make(map[int]bool)
	for _, faceID := range visibleFaces {
		uniqueFaces[faceID] = true
	}
	return uniqueFaces
}

// getFaceIdsFromUniqueFaces converts a face set to a slice
func getFaceIdsFromUniqueFaces(uniqueFaces map[int]bool) []int {
	clusterFaces := make([]int, 0, len(uniqueFaces))
	for faceID := range uniqueFaces {
		clusterFaces = append(clusterFaces, faceID)
	}
	return clusterFaces
}

// getTreeLeaves creates the final tree structure with visibility data
func getTreeLeaves(mapData *q2file.MapData, allLeaves []TreeLeaf, facesFromCluster map[ClusterID][]int, allFaceIds []int) []TreeLeaf {
	newLeafFaces := make([]TreeLeaf, len(allLeaves))
	bspLeaves := mapData.BSPLeaves

	for i := range allLeaves {
		clusterID := ClusterID(bspLeaves[i].Cluster)
		if clusterID != ClusterID(ClusterInvalidID) {
			newLeafFaces[i] = TreeLeaf{
				LeafIndex: i,
				Faces:     facesFromCluster[clusterID],
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

// FindLeafNode finds the leaf node containing the given position
func (tree *BSPTree) FindLeafNode(startNode int, mapData *q2file.MapData, position [3]float32) TreeLeaf {
	var d float32
	nodeID := startNode

	// Traverse the BSP tree until we reach a leaf
	for nodeID >= 0 {
		node := mapData.Nodes[nodeID]
		plane := mapData.Planes[node.Plane]

		// Calculate distance to plane
		if plane.Type < 3 {
			// Axis-aligned plane
			d = position[plane.Type] - plane.Distance
		} else {
			// Arbitrary plane
			dotProduct := position[0]*plane.Normal[0] +
				position[1]*plane.Normal[1] +
				position[2]*plane.Normal[2]
			d = dotProduct - plane.Distance
		}

		// Determine which side of the plane the position is on
		if d < 0 {
			nodeID = int(node.BackChild)
		} else {
			nodeID = int(node.FrontChild)
		}
	}

	// Convert negative node ID to leaf index
	return tree.TreeLeaves[-(nodeID + 1)]
}
