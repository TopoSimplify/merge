package merge

import (
	"sort"
	"simplex/rng"
	"simplex/node"
	"simplex/lnr"
	"simplex/knn"
	"github.com/intdxdt/sset"
	"github.com/intdxdt/rtree"
	"github.com/intdxdt/geom"
)

func sortInts(iter []int) []int {
	sort.Ints(iter)
	return iter
}

//node.Nodes from Rtree boxes
func nodesFromBoxes(iter []rtree.BoxObj) *node.Nodes {
	var self = node.NewNodes(len(iter))
	for _, h := range iter {
		self.Push(h.(*node.Node))
	}
	return self
}

//Merge two ranges
func Range(ra, rb *rng.Range) *rng.Range {
	var ranges = sortInts(append(ra.AsSlice(), rb.AsSlice()...))
	// i...[ra]...k...[rb]...j
	return rng.NewRange(ranges[0], ranges[len(ranges)-1])
}

//Merge contiguous fragments based combined score
func ContiguousFragmentsAtThreshold(self lnr.Linear, ha, hb *node.Node,
	scoreRelation func(float64) bool, gfn geom.GeometryFn) *node.Node {
	_, val := self.Score(self, Range(ha.Range, hb.Range))
	if scoreRelation(val) {
		return ContiguousFragments(self, ha, hb, gfn)
	}
	return nil
}

//Merge contiguous hulls
func ContiguousFragments(self lnr.Linear, ha, hb *node.Node, gfn geom.GeometryFn) *node.Node {
	var r = Range(ha.Range, hb.Range)
	// i...[ha]...k...[hb]...j
	return node.New(self.Polyline(), r, gfn)
}

//Merge contiguous hulls by fragment size
func ContiguousFragmentsBySize(self lnr.Linear,
	hulls []*node.Node, hulldb *rtree.RTree, vertexSet *sset.SSet,
	unmerged map[[2]int]*node.Node, fragmentSize int,
	isScoreValid func(float64) bool, gfn geom.GeometryFn, EpsilonDist float64,
) ([]*node.Node, []*node.Node) {

	//@formatter:off
	var pln       = self.Polyline()
	var keep      = make([]*node.Node, 0)
	var rm        = make([]*node.Node, 0)

	var hdict    = make(map[[2]int]*node.Node, 0)
	var mrgdict  = make(map[[2]int]*node.Node, 0)

	var isMerged = func(o *rng.Range) bool {
		_, ok := mrgdict[o.AsArray()]
		return ok
	}

	for _, h := range hulls {
		hr := h.Range

		if isMerged(hr){
			continue
		}

		hdict[h.Range.AsArray()] = h

		if hr.Size() != fragmentSize {
			continue
		}

		// sort hulls for consistency
		var hs = nodesFromBoxes(knn.FindNodeNeighbours(hulldb, h, EpsilonDist)).Sort()

		for _, s := range hs.DataView() {
			sr := s.Range
			if isMerged(sr){
				continue
			}

			//merged range
			r := Range(sr, hr)

			//test whether sr.i or sr.j is a self inter-vertex -- split point
			//not sr.i != hr.i or sr.j != hr.j without i/j being a inter-vertex
			//tests for contiguous and whether contiguous index is part of vertex set
			//if the location at which they are contiguous is not part of vertex set then
			//its mergeable : mergeable score <= threshold
			mergeable := (hr.J() == sr.I() && !vertexSet.Contains(sr.I())) ||
				         (hr.I() == sr.J() && !vertexSet.Contains(sr.J()))

			if mergeable {
				_, val      := self.Score(self, r)
				mergeable   = isScoreValid(val)
			}

			if !mergeable {
				unmerged[hr.AsArray()] = h
				continue
			}

			//keep track of items merged
			mrgdict[hr.AsArray()] = h
			mrgdict[sr.AsArray()] = s

			// rm sr + hr
			delete(hdict, sr.AsArray())
			delete(hdict, hr.AsArray())

			// add merge
			hdict[r.AsArray()] = node.New(pln, r, gfn)

			// add to remove list to remove , after merge
			rm = append(rm, s)
			rm = append(rm, h)

			//if present in umerged as fragment remove
			delete(unmerged, hr.AsArray())
			break
		}
	}

	for _, o := range hdict {
		keep = append(keep, o)
	}
	return keep, rm
}
