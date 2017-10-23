package merge

import (
	"sort"
	"simplex/dp"
	"simplex/lnr"
	"simplex/knn"
	"simplex/node"
	"github.com/intdxdt/sset"
	"github.com/intdxdt/rtree"
)

//Merge segment fragments where possible
func SimpleSegments(self lnr.Linear, nodedb *rtree.RTree,
	constVertexSet *sset.SSet, scoreRelation scoreRelationFn,
	validateMerge func(node *node.Node, nodedb *rtree.RTree) bool) {

	var hull *node.Node
	var fragmentSize = 1
	var neighbours *node.Nodes
	var cache = make(map[[4]int]bool)
	var hulls = nodesFromRtreeNodes(nodedb.All()).Sort().AsDeque()

	//fmt.Println("After constraints:")
	//DebugPrintPtSet(hulls)

	for !hulls.IsEmpty() {
		// from left
		hull = popLeftHull(hulls)

		if hull.Range.Size() != fragmentSize {
			continue
		}

		//make sure hull index is not part of vertex with degree > 2
		if constVertexSet.Contains(hull.Range.I()) || constVertexSet.Contains(hull.Range.J()) {
			continue
		}

		nodedb.Remove(hull)

		// find context neighbours
		neighbours = nodesFromBoxes(knn.FindNodeNeighbours(nodedb, hull, knn.EpsilonDist))

		// find context neighbours
		prev, nxt := node.Neighbours(hull, neighbours)

		// find mergeable neighbours contiguous 
		var key [4]int
		var mergePrev, mergeNxt *node.Node

		if prev != nil {
			key = cacheKey(prev, hull)
			if !cache[key] {
				addToMergeCache(cache, &key)
				mergePrev = ContiguousFragmentsAtThreshold(self, prev, hull, scoreRelation, dp.NodeGeometry)
			}
		}

		if nxt != nil {
			key = cacheKey(hull, nxt)
			if !cache[key] {
				addToMergeCache(cache, &key)
				mergeNxt = ContiguousFragmentsAtThreshold(self, hull, nxt, scoreRelation, dp.NodeGeometry)
			}
		}

		var merged bool
		//nxt, prev
		if !merged && mergeNxt != nil {
			nodedb.Remove(nxt)
			if validateMerge(mergeNxt, nodedb) {
				var h = castAsNode(hulls.First())
				if h == nxt {
					hulls.PopLeft()
				}
				nodedb.Insert(mergeNxt)
				merged = true
			} else {
				merged = false
				nodedb.Insert(nxt)
			}
		}

		if !merged && mergePrev != nil {
			nodedb.Remove(prev)
			//prev cannot exist since moving from left --- right
			if validateMerge(mergePrev, nodedb) {
				nodedb.Insert(mergePrev)
				merged = true
			} else {
				merged = false
				nodedb.Insert(prev)
			}
		}

		if !merged {
			nodedb.Insert(hull)
		}
	}
}

func cacheKey(a, b *node.Node) [4]int {
	var ij = [4]int{a.Range.I(), a.Range.J(), b.Range.I(), b.Range.J()}
	sort.Ints(ij[:])
	return ij
}

func addToMergeCache(cache map[[4]int]bool, key *[4]int) {
	cache[*key] = true
}
