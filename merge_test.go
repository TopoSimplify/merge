package merge

import (
	"time"
	"testing"
	"simplex/dp"
	"simplex/node"
	"simplex/opts"
	"simplex/split"
	"simplex/offset"
	"github.com/intdxdt/cmp"
	"github.com/intdxdt/sset"
	"github.com/franela/goblin"
	"github.com/intdxdt/rtree"
)

const epsilonDist = 1.0e-5

//@formatter:off
func TestMergeNode(t *testing.T) {
	g := goblin.Goblin(t)
	g.Describe("test merge hull", func() {
		g.It("should test merge", func() {

			g.Timeout(1 * time.Hour)
			options := &opts.Opts{
				Threshold:              50.0,
				MinDist:                20.0,
				RelaxDist:              30.0,
				KeepSelfIntersects:     true,
				AvoidNewSelfIntersects: true,
				GeomRelation:           true,
				DistRelation:           false,
				DirRelation:            false,
			}
			//checks if score is valid at threshold of constrained dp
			var isScoreRelateValid = func(val float64) bool {
				return val <= options.Threshold
			}

			// self.relates = relations(self)
			var wkt = "LINESTRING ( 860 390, 810 360, 770 400, 760 420, 800 440, 810 470, 850 500, 810 530, 780 570, 760 530, 720 530, 710 500, 650 450 )"
			var coords = linearCoords(wkt)
			var n = len(coords) - 1
			var homo = dp.New(coords, options, offset.MaxOffset)

			var hull = createHulls([][]int{{0, n}}, coords)[0]
			var ha, hb = split.AtScoreSelection(hull, homo.Score, hullGeom)
			var splits = split.AtIndex(hull, []int{
				ha.Range.I(), ha.Range.J(), hb.Range.I(),
				hb.Range.I() - 1, hb.Range.J(),
			}, hullGeom)

			g.Assert(len(splits)).Equal(3)

			var hulldb = rtree.NewRTree(8)
			var boxes = make([]rtree.BoxObj, len(splits))
			for i, v := range splits {
				boxes[i] = v
			}
			hulldb.Load(boxes)

			vertex_set := sset.NewSSet(cmp.Int)
			var unmerged = make(map[[2]int]*node.Node, 0)

			var keep, rm = ContiguousFragmentsBySize(
				splits, hulldb, vertex_set, unmerged, 1,
				isScoreRelateValid, homo.Score, hullGeom, epsilonDist)

			g.Assert(len(keep)).Equal(2)
			g.Assert(len(rm)).Equal(2)

			splits = split.AtIndex(hull, []int{0, 5, 6, 7, 8, 12}, hullGeom)
			g.Assert(len(splits)).Equal(5)

			hulldb = rtree.NewRTree(8)
			boxes = make([]rtree.BoxObj, len(splits))
			for i, v := range splits {boxes[i] = v}
			hulldb.Load(boxes)

			vertex_set = sset.NewSSet(cmp.Int)
			unmerged = make(map[[2]int]*node.Node, 0)

			keep, rm = ContiguousFragmentsBySize(
				splits, hulldb, vertex_set, unmerged, 1,
				isScoreRelateValid, homo.Score, hullGeom, epsilonDist)

			g.Assert(len(keep)).Equal(3)
			g.Assert(len(rm)).Equal(4)
		})
	})
}
