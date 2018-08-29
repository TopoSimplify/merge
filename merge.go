package merge

import (
	"sort"
	"github.com/TopoSimplify/rng"
	"github.com/TopoSimplify/node"
	"github.com/TopoSimplify/lnr"
	"github.com/TopoSimplify/knn"
	"github.com/TopoSimplify/common"
	"github.com/intdxdt/geom"
	"github.com/TopoSimplify/hdb"
	"github.com/intdxdt/iter"
)

//Merge two ranges
func Range(ra, rb rng.Rng) rng.Rng {
	var ranges = common.SortInts(append(ra.AsSlice(), rb.AsSlice()...))
	// i...[ra]...k...[rb]...j
	return rng.Range(ranges[0], ranges[len(ranges)-1])
}

//Merge contiguous fragments based combined score
func ContiguousFragmentsAtThreshold(
	id *iter.Igen, scoreFn lnr.ScoreFn, ha, hb *node.Node,
	scoreRelation func(float64) bool, gfn func(geom.Coords)geom.Geometry) (bool, node.Node) {

	if !ha.Range.Contiguous(hb.Range) {
		panic("node are not contiguous")
	}

	var coordinates = ContiguousCoordinates(ha, hb)
	var _, val = scoreFn(coordinates)
	if scoreRelation(val) {
		return true, contiguousFragments(id, coordinates, ha, hb, gfn)
	}
	return false, node.Node{}
}

func ContiguousCoordinates(ha, hb *node.Node) geom.Coords {
	if !ha.Range.Contiguous(hb.Range) {
		panic("node are not contiguous")
	}

	if hb.Range.I < ha.Range.J && hb.Range.J == ha.Range.I {
		ha, hb = hb, ha
	}

	var coordinates = ha.Coordinates()
	var n = coordinates.Len() - 1
	coordinates.Idxs = append(coordinates.Idxs[:n:n], hb.Coordinates().Idxs...)
	return coordinates
}

//Merge contiguous hulls
func contiguousFragments(id *iter.Igen, coordinates geom.Coords, ha, hb *node.Node,
	gfn func(geom.Coords)geom.Geometry) node.Node {
	// i...[ha]...k...[hb]...j
	return node.CreateNode(id, coordinates, Range(ha.Range, hb.Range), gfn, ha.Instance)
}

//Merge contiguous hulls by fragment size
func ContiguousFragmentsBySize(
	id *iter.Igen,
	hulls []node.Node,
	hulldb *hdb.Hdb,
	vertexSet map[int]bool,
	unmerged map[[2]int]*node.Node,
	fragmentSize int,
	isScoreValid func(float64) bool,
	scoreFn lnr.ScoreFn,
	gfn func(geom.Coords)geom.Geometry,
	EpsilonDist float64) ([]*node.Node, []*node.Node) {

	//@formatter:off
	var keep = make([]*node.Node, 0)
	var rm = make([]*node.Node, 0)

	var hdict = make(map[[2]int]*node.Node, 0)
	var mrgdict = make(map[[2]int]*node.Node, 0)

	var isMerged = func(o rng.Rng) bool {
		_, ok := mrgdict[o.AsArray()]
		return ok
	}

	for i := range hulls {
		var h = &hulls[i]
		var hr = h.Range
		if isMerged(hr) {
			continue
		}

		hdict[h.Range.AsArray()] = h

		if hr.Size() != fragmentSize {
			continue
		}

		// sort hulls for consistency
		var hs = knn.FindNodeNeighbours(hulldb, h, EpsilonDist)
		sort.Sort(node.NodePtrs(hs))

		for i := range hs {
			s := hs[i]
			sr := s.Range
			if isMerged(sr) {
				continue
			}

			//test whether sr.i or sr.j is a self inter-vertex -- split point
			//not sr.i != hr.i or sr.j != hr.j without i/j being a inter-vertex
			//tests for contiguous and whether contiguous index is part of vertex set
			//if the location at which they are contiguous is not part of vertex set then
			//its mergeable : mergeable score <= threshold
			var mergeable = (hr.J == sr.I && !vertexSet[sr.I]) || (hr.I == sr.J && !vertexSet[sr.J])

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

			var _nd = node.CreateNode(id, coords, r, gfn, h.Instance)
			// add merge
			hdict[r.AsArray()] = &_nd

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
