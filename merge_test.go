package merge

import (
	"time"
	"testing"
	"simplex/opts"
	"simplex/offset"
	"github.com/intdxdt/rtree"
	"github.com/franela/goblin"
	"simplex/split"
	"simplex/dp"
	"github.com/intdxdt/sset"
	"github.com/intdxdt/cmp"
	"simplex/node"
)

const epsilonDist = 1.0e-5

//@formatter:off
func TestMergeHull(t *testing.T) {
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
			var is_score_relate_valid = func (val float64) bool {
				return val <= options.Threshold
			}

			// self.relates = relations(self)
			wkt     := "LINESTRING ( 860 390, 810 360, 770 400, 760 420, 800 440, 810 470, 850 500, 810 530, 780 570, 760 530, 720 530, 710 500, 650 450 )"
			coords  := linear_coords(wkt)
			n       := len(coords) - 1
			homo    := dp.New(coords,  options, offset.MaxOffset)

			hull    := create_hulls([][]int{{0, n}}, coords)[0]
			ha, hb  := split.AtScoreSelection(homo, hull, hullGeom)
			splits  := split.AtIndex(homo, hull, []int{
				ha.Range.I(), ha.Range.J(), hb.Range.I(),
				hb.Range.I() - 1, hb.Range.J(),
			}, hullGeom)
			g.Assert(len(splits)).Equal(3)

			hulldb := rtree.NewRTree(8)
			boxes := make([]rtree.BoxObj, len(splits))
			for i, v := range splits {
				boxes[i] = v
			}
			hulldb.Load(boxes)

			vertex_set := sset.NewSSet(cmp.Int)
			var unmerged = make(map[[2]int]*node.Node,0)

			keep, rm := ContiguousFragmentsBySize(
				homo, splits, hulldb, vertex_set, unmerged, 1,
				is_score_relate_valid, hullGeom, epsilonDist)

			g.Assert(len(keep)).Equal(2)
			g.Assert(len(rm)).Equal(2)

			splits  = split.AtIndex(homo, hull, []int{0, 5, 6, 7, 8, 12}, hullGeom)
			g.Assert(len(splits)).Equal(5)

			hulldb  = rtree.NewRTree(8)
			boxes   = make([]rtree.BoxObj, len(splits))
			for i, v := range splits {boxes[i] = v}
			hulldb.Load(boxes)

			vertex_set = sset.NewSSet(cmp.Int)
			unmerged = make(map[[2]int]*node.Node,0)

			keep, rm = ContiguousFragmentsBySize(
				homo, splits, hulldb, vertex_set, unmerged, 1,
				is_score_relate_valid, hullGeom, epsilonDist)

			g.Assert(len(keep)).Equal(3)
			g.Assert(len(rm)).Equal(4)
		})
	})
}
