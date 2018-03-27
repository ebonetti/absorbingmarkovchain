package absorbingmarkovchain

import (
	"github.com/RoaringBitmap/roaring"
)

type dGraph struct {
	Nodes *roaring.Bitmap
	Edges func(from uint32) (to []uint32)
}

func (gin dGraph) addSelfLoops() (gout dGraph) {
	gout.Edges = func(from uint32) (to []uint32) {
		to = gin.Edges(from)
		if match, p := uint32Exist(to, from); !match {
			to = append(append(append(make([]uint32, 0, len(to)+1), to[:p]...), from), to[p:]...)
		}
		return
	}
	gout.Nodes = gin.Nodes

	return
}

func (gin dGraph) filterNodes(blacklist *roaring.Bitmap) (gout dGraph) {
	blacklistArray := blacklist.ToArray()
	gout.Edges = func(from uint32) (to []uint32) {
		if blacklist.Contains(from) {
			return nil
		}

		to = gin.Edges(from)
		u, up := to, 0                            //unfiltered to and unfiltered to position
		b, bp, bv := blacklistArray, 0, uint32(0) //blacklistArray, position in blacklistArray and value
		match := false
		for bp, bv = range b {
			match, up = uint32Exist(u, bv)
			u = u[up:]
			if match {
				break
			}
		}
		if !match {
			return to
		}
		if len(u) == 1 {
			return to[:len(to)-1]
		}

		p := len(to) - len(u)
		to = append([]uint32{}, to...)[:p:p]
		u = u[1:]

		bp++
		b = b[bp:]
		for _, uv := range u {
			match, bp = uint32Exist(b, uv)
			b = b[bp:]
			if !match {
				to = append(to, uv)
			}
		}
		return
	}

	gout.Nodes = roaring.AndNot(gin.Nodes, blacklist)

	return
}

func (gin dGraph) normalizedIDs() (gout dGraph, t translator) {
	new2OldID := gin.Nodes.ToArray()
	l := len(new2OldID)
	gout.Edges = func(from uint32) (to []uint32) {
		new2OldID := new2OldID
		to = append([]uint32{}, gin.Edges(new2OldID[from])...)
		for p, oldID := range to {
			new2OldID = new2OldID[uint32Search(new2OldID, oldID):]
			to[p] = uint32(l - len(new2OldID))
		}
		return
	}

	gout.Nodes = roaring.NewBitmap()
	gout.Nodes.AddRange(0, uint64(len(new2OldID)))

	t = myTranslator(new2OldID)

	return
}

type wDGraph struct {
	dGraph
	Weighter func(from, to uint32) (weight float64, err error)
}

func (gin wDGraph) normalizedWeights() (gout wDGraph, err error) {
	nodeCount := uint32(gin.Nodes.GetCardinality())
	weightSum := make([]float64, 0, nodeCount)
	weights := make([]float64, 0, 1024)
	for i := gin.Nodes.Iterator(); i.HasNext(); {
		from := i.Next()
		weights = weights[:0]
		for _, to := range gin.Edges(from) {
			w, err := gin.Weighter(from, to)
			if err != nil {
				return gout, err
			}
			weights = append(weights, w)
		}
		weightSum = append(weightSum, fsum(weights))
	}

	old2NewID := func(n uint32) (uint32, error) { //Identity function
		return n, nil
	}
	if m := nodeCount - 1; !gin.Nodes.Contains(m) || uint32(gin.Nodes.Rank(m)) != nodeCount { //m is not max of a set of type [0,n]
		old2NewID = newTranslator(gin.Nodes).ToNew
	}

	gout.Weighter = func(from, to uint32) (weight float64, err error) {
		if weight, err = gin.Weighter(from, to); err != nil {
			return
		}
		newID, err := old2NewID(from)
		if err != nil {
			return
		}
		weight /= weightSum[newID]

		return
	}

	gout.dGraph = gin.dGraph

	return
}

func (gin wDGraph) addSelfLoops() (gout wDGraph) {
	gout.Weighter = func(from, to uint32) (weight float64, err error) {
		weight, err = gin.Weighter(from, to)
		if from != to {
			return
		}
		match, _ := uint32Exist(gin.Edges(from), from)
		switch {
		case match && err != nil:
			//do nothing
		case match:
			weight += -1
		default:
			weight = -1
		}

		return
	}

	gout.dGraph = gin.dGraph.addSelfLoops()

	return
}

func (gin wDGraph) normalizedIDs() (gout wDGraph, t translator) {
	gout.dGraph, t = gin.dGraph.normalizedIDs()

	gout.Weighter = func(from, to uint32) (weight float64, err error) {
		if to, err = t.ToOld(to); err != nil {
			return
		}
		if from, err = t.ToOld(from); err != nil {
			return
		}
		if weight, err = gin.Weighter(from, to); err != nil {
			return
		}
		return
	}

	return
}
