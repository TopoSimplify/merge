package merge

import (
	"simplex/node"
	"github.com/intdxdt/rtree"
	"github.com/intdxdt/deque"
)

type scoreRelationFn func(float64) bool

func popLeftHull(que *deque.Deque) *node.Node {
	return que.PopLeft().(*node.Node)
}

func castAsNode(o interface{}) *node.Node {
	return o.(*node.Node)
}

//node.Nodes from Rtree nodes
func nodesFromRtreeNodes(iter []*rtree.Node) *node.Nodes {
	var self = node.NewNodes(len(iter))
	for _, h := range iter {
		self.Push(h.GetItem().(*node.Node))
	}
	return self
}
