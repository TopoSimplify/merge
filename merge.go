package merge

import (
	"sort"
	"simplex/rng"
	"simplex/node"
	"simplex/lnr"
	"github.com/intdxdt/geom"
)

type isScoreRelateValid func(float64) bool

func sortInts(iter []int) []int {
	sort.Ints(iter)
	return iter
}

//Merge two ranges
func Range(ra, rb *rng.Range) *rng.Range {
	var ranges = sortInts(append(ra.AsSlice(), rb.AsSlice()...))
	// i...[ra]...k...[rb]...j
	return rng.NewRange(ranges[0], ranges[len(ranges)-1])
}

//Merge contiguous fragments based combined score
func ContiguousFragmentsAtThreshold(self lnr.Linear, ha, hb *node.Node,
	isScoreValid isScoreRelateValid, gfn geom.GeometryFn) *node.Node {
	_, val := self.Score(self, Range(ha.Range, hb.Range))
	if isScoreValid(val) {
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
