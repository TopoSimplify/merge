package merge

import (
	"time"
	"testing"
	"github.com/TopoSimplify/dp"
	"github.com/TopoSimplify/node"
	"github.com/TopoSimplify/opts"
	"github.com/TopoSimplify/split"
	"github.com/TopoSimplify/offset"
	"github.com/TopoSimplify/common"
	"github.com/franela/goblin"
	"github.com/TopoSimplify/hdb"
	"github.com/intdxdt/iter"
)

const epsilonDist = 1.0e-5

var hullGeom = common.Geometry
var linearCoords = common.LinearCoords
var createHulls = common.CreateHulls

//@formatter:off
func TestMergeNode(t *testing.T) {
	var g = goblin.Goblin(t)
	var id = iter.NewIgen()
	g.Describe("test merge hull", func() {
		g.It("should test merge at threshold", func() {
			//checks if score is valid at threshold of constrained dp
			var coords = linearCoords("LINESTRING ( 960 840, 980 840, 980 880, 1020 900, 1080 880, 1120 860, 1160 800, 1160 760, 1140 700, 1080 700, 1040 720, 1060 760, 1120 800, 1080 840, 1020 820, 940 760 )")
			var hulls = createHulls(id,
				[][]int{{0, 2}, {2, 6}, {6, 8}, {8, 10}, {10, 12}, {12, coords.Len() - 1}}, coords, nil)
			var gfn = hullGeom
			var bln, n = ContiguousFragmentsAtThreshold(
				id, offset.MaxOffset,
				&hulls[0], &hulls[1],
				func(val float64) bool { return val <= 50.0 }, gfn,
			)

			g.Assert(bln).IsFalse()
			g.Assert(n == node.Node{})

			bln, n = ContiguousFragmentsAtThreshold(
				id, offset.MaxOffset,
				&hulls[0], &hulls[1],
				func(val float64) bool { return val <= 100.0 }, gfn,
			)
			g.Assert(bln).IsTrue()

			g.Assert(ContiguousCoordinates(&hulls[0], &hulls[1])).Equal(
				coords.Slice(0, hulls[1].Range.J+1),
			)
			g.Assert(ContiguousCoordinates(&hulls[2], &hulls[1])).Equal(
				coords.Slice(hulls[1].Range.I, hulls[2].Range.J+1),
			)
		})

		g.It("should test merge non contiguous", func() {
			defer func() {
				g.Assert(recover() != nil)
			}()
			//checks if score is valid at threshold of constrained dp
			var coords = linearCoords("LINESTRING ( 960 840, 980 840, 980 880, 1020 900, 1080 880, 1120 860, 1160 800, 1160 760, 1140 700, 1080 700, 1040 720, 1060 760, 1120 800, 1080 840, 1020 820, 940 760 )")
			var hulls = createHulls(id,
				[][]int{{0, 2}, {2, 6}, {6, 8}, {8, 10}, {10, 12}, {12, coords.Len() - 1}}, coords, nil)
			var gfn = hullGeom
			ContiguousFragmentsAtThreshold(id, offset.MaxOffset,
				&hulls[0], &hulls[2],
				func(val float64) bool { return val <= 100.0 }, gfn,
			)

		})
		g.It("should test merge non contiguous coords", func() {
			defer func() {
				g.Assert(recover() != nil)
			}()
			//checks if score is valid at threshold of constrained dp
			var coords = linearCoords("LINESTRING ( 960 840, 980 840, 980 880, 1020 900, 1080 880, 1120 860, 1160 800, 1160 760, 1140 700, 1080 700, 1040 720, 1060 760, 1120 800, 1080 840, 1020 820, 940 760 )")
			var hulls = createHulls(id,
				[][]int{{0, 2}, {2, 6}, {6, 8}, {8, 10}, {10, 12}, {12, coords.Len() - 1}}, coords, nil)
			ContiguousCoordinates(&hulls[0], &hulls[2])
		})

		g.It("should test merge", func() {
			g.Timeout(1 * time.Hour)
			options := &opts.Opts{
				Threshold:              50.0,
				MinDist:                20.0,
				RelaxDist:              30.0,
				PlanarSelf:             true,
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
			var n = coords.Len() - 1
			var homo = dp.New(id.Next(), coords, options, offset.MaxOffset)
			var hull = createHulls(id, [][]int{{0, n}}, coords, homo)[0]
			var ha, hb = split.AtScoreSelection(id, &hull, homo.Score, hullGeom)
			var splits = split.AtIndex(id, &hull, []int{
				ha.Range.I, ha.Range.J, hb.Range.I, hb.Range.I - 1, hb.Range.J,
			}, hullGeom)

			g.Assert(len(splits)).Equal(3)

			var hulldb = hdb.NewHdb().Load(splits)

			var vertexSet = make(map[int]bool)
			var unmerged = make(map[[2]int]*node.Node, 0)

			var keep, rm = ContiguousFragmentsBySize(
				id, splits, hulldb, vertexSet, unmerged, 1,
				isScoreRelateValid, homo.Score, hullGeom, epsilonDist)

			g.Assert(len(keep)).Equal(2)
			g.Assert(len(rm)).Equal(2)

			splits = split.AtIndex(id, &hull, []int{0, 5, 6, 7, 8, 12}, hullGeom)
			g.Assert(len(splits)).Equal(5)

			hulldb = hdb.NewHdb().Load(splits)

			vertexSet = make(map[int]bool)
			unmerged = make(map[[2]int]*node.Node, 0)

			keep, rm = ContiguousFragmentsBySize(
				id, splits, hulldb, vertexSet, unmerged, 1,
				isScoreRelateValid, homo.Score, hullGeom, epsilonDist)

			g.Assert(len(keep)).Equal(3)
			g.Assert(len(rm)).Equal(4)
		})
	})
}
