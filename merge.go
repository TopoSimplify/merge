package merge

import (
	"simplex/rng"
	"simplex/node"
	"simplex/lnr"
	"simplex/knn"
	"simplex/common"
	"github.com/intdxdt/sset"
	"github.com/intdxdt/rtree"
	"github.com/intdxdt/geom"
	"sort"
)

//Merge two ranges
func Range(ra, rb *rng.Range) *rng.Range {
	var ranges = common.SortInts(append(ra.AsSlice(), rb.AsSlice()...))
	// i...[ra]...k...[rb]...j
	return rng.NewRange(ranges[0], ranges[len(ranges)-1])
}

//Merge contiguous fragments based combined score
func ContiguousFragmentsAtThreshold(
	scoreFn lnr.ScoreFn, ha, hb *node.Node,
	scoreRelation func(float64) bool, gfn geom.GeometryFn,
) *node.Node {
	if !ha.Range.Contiguous(hb.Range) {
		panic("node are not contiguous")
	}
	var coordinates = ContiguousCoordinates(ha, hb)
	_, val := scoreFn(coordinates)
	if scoreRelation(val) {
		return contiguousFragments(coordinates, ha, hb, gfn)
	}
	return nil
}

func ContiguousCoordinates(ha, hb *node.Node) []*geom.Point {
	if !ha.Range.Contiguous(hb.Range) {
		panic("node are not contiguous")
	}
	//var nodes = []*node.Node{ha, hb}
	//sort.Sort(node.Nodes(nodes))
	if hb.Range.I < ha.Range.J && hb.Range.J == ha.Range.I {
		ha, hb = hb, ha
	}

	var coordinates = ha.Coordinates()
	var n = len(coordinates) - 1
	coordinates = append(coordinates[:n:n], hb.Coordinates()...)
	return coordinates
}

//Merge contiguous hulls
func contiguousFragments(
	coordinates []*geom.Point,
	ha, hb *node.Node,
	gfn geom.GeometryFn,
) *node.Node {
	var r = Range(ha.Range, hb.Range)
	// i...[ha]...k...[hb]...j
	return node.New(coordinates, r, gfn)
}

//Merge contiguous hulls by fragment size
func ContiguousFragmentsBySize(
	hulls []*node.Node,
	hulldb *rtree.RTree,
	vertexSet *sset.SSet,
	unmerged map[[2]int]*node.Node,
	fragmentSize int,
	isScoreValid func(float64) bool,
	scoreFn lnr.ScoreFn,
	gfn geom.GeometryFn,
	EpsilonDist float64,
) ([]*node.Node, []*node.Node) {

	//@formatter:off
	var keep = make([]*node.Node, 0)
	var rm = make([]*node.Node, 0)

	var hdict = make(map[[2]int]*node.Node, 0)
	var mrgdict = make(map[[2]int]*node.Node, 0)

	var isMerged = func(o *rng.Range) bool {
		_, ok := mrgdict[o.AsArray()]
		return ok
	}

	for _, h := range hulls {
		hr := h.Range

		if isMerged(hr) {
			continue
		}

		hdict[h.Range.AsArray()] = h

		if hr.Size() != fragmentSize {
			continue
		}

		// sort hulls for consistency
		var hs = common.NodesFromBoxes(knn.FindNodeNeighbours(hulldb, h, EpsilonDist))
		sort.Sort(node.Nodes(hs))

		for _, s := range hs {
			sr := s.Range
			if isMerged(sr) {
				continue
			}

			//test whether sr.i or sr.j is a self inter-vertex -- split point
			//not sr.i != hr.i or sr.j != hr.j without i/j being a inter-vertex
			//tests for contiguous and whether contiguous index is part of vertex set
			//if the location at which they are contiguous is not part of vertex set then
			//its mergeable : mergeable score <= threshold
			var mergeable = (
				(hr.J == sr.I && !vertexSet.Contains(sr.I)) ||
					(hr.I == sr.J && !vertexSet.Contains(sr.J)))

			if mergeable {
				var _, val = scoreFn(ContiguousCoordinates(s, h))
				mergeable = isScoreValid(val)
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
			//merged range
			var coords, r = ContiguousCoordinates(h, s), Range(sr, hr)
			// add merge
			hdict[r.AsArray()] = node.New(coords, r, gfn)

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
