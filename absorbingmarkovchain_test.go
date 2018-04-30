package absorbingmarkovchain

import (
	"context"
	"testing"

	"github.com/RoaringBitmap/roaring"
)

func TestAbsorptionProbabilities(t *testing.T) {
	chain, tn2anw := amcSample()
	weighter, err := chain.AbsorptionProbabilities(context.Background())
	if err != nil {
		t.Error(err)
	}
	const eps = 1.e-15
	for tn, nodes := range tn2anw {
		for _, node := range nodes {
			w, err := weighter(tn, node.to)
			switch {
			case err != nil:
				t.Error(err)
			case (w-node.w)*(w-node.w) > eps*eps:
				t.Errorf("The assignment probability in edge (%v,%v) is %v while is evaluated as %v", tn, node.to, node.w, w)
			}
		}
	}
}
func TestAbsorptionAssignments(t *testing.T) {
	chain, tn2anw := amcSample()
	assigner, err := chain.AbsorptionAssignments(context.Background())
	if err != nil {
		t.Error(err)
	}
	for tn, nodes := range tn2anw {
		an, ok := assigner[tn]
		switch {
		case !ok:
			t.Errorf("%v is not a valid node, but it should be.", tn)
		case an != nodes[0].to:
			t.Errorf("%v should be assigned to %v, but it's assigned to %v", tn, nodes[0].to, an)
		}
	}
}

func amcSample() (chain *AbsorbingMarkovChain, tn2anw map[uint32][]implicitWeightedEdge) {
	m := map[uint32][]uint32{2: {0, 4}, 3: {1, 4}, 4: {0, 1, 2}, 5: {3}, 6: {2, 4}, 7: {1, 3, 4}}
	//edges are sorted by descending weight
	tn2anw = map[uint32][]implicitWeightedEdge{2: {{0, 0.8}, {1, 0.2}}, 3: {{1, 0.7}, {0, 0.3}}, 4: {{0, 0.6}, {1, 0.4}}, 5: {{1, 0.7}, {0, 0.3}}, 6: {{0, 0.7}, {1, 0.3}}, 7: {{1, 0.7}, {0, 0.3}}}
	nodes := roaring.NewBitmap()
	for from, to := range m {
		nodes.Add(from)
		for _, to := range to {
			nodes.Add(to)
		}
	}
	absorbingNodes := roaring.NewBitmap()
	for i := nodes.Iterator(); i.HasNext(); {
		from := i.Next()
		to := m[from]
		switch {
		case len(to) == 0:
			fallthrough
		case len(to) == 1 && from == to[0]:
			absorbingNodes.Add(from)
		}
	}

	chain = &AbsorbingMarkovChain{
		wDGraph{
			dGraph{
				nodes,
				func(from uint32) []uint32 { return m[from] },
			},
			func(from, to uint32) (weight float64, err error) { return 1, nil },
		},
		absorbingNodes,
		"",
	}
	return
}
