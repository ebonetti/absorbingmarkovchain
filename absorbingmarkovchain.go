// Package absorbingmarkovchain provides primitives for computing absorption probabilities of absorbing markov chains.
package absorbingmarkovchain

import (
	"context"
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/RoaringBitmap/roaring"
	"github.com/pkg/errors"

	"github.com/ebonetti/absorbingmarkovchain/internal/gmres"
)

// New creates a new absorbing markov chain.
func New(tmpDir string, nodes, absorbingNodes *roaring.Bitmap, edges func(from uint32) (to []uint32), weighter func(from, to uint32) (weight float64, err error)) *AbsorbingMarkovChain {
	return &AbsorbingMarkovChain{
		wDGraph{
			dGraph{
				nodes,
				edges,
			},
			weighter,
		},
		absorbingNodes,
		tmpDir,
	}
}

// AbsorbingMarkovChain represents an absorbing markov chain.
type AbsorbingMarkovChain struct {
	wDGraph
	absorbingNodes *roaring.Bitmap
	tmpDir         string
}

// AbsorptionProbabilities calculates absorption probabilities for the current absorbing markov chain.
func (chain *AbsorbingMarkovChain) AbsorptionProbabilities(ctx context.Context) (weighter func(from, to uint32) (weight float64, err error), err error) {
	fuzzyAssignments, ttn, tan, err := chain.absorptionProbabilities(ctx, func() { chain = nil }) //enable eventual GC
	if err != nil {
		return
	}

	return func(from, to uint32) (weight float64, err error) {
		a, e1 := tan.ToNew(to)
		t, e2 := ttn.ToNew(from)
		switch {
		case e1 != nil:
			err = e1
		case e2 != nil:
			err = e2
		default:
			weight = fuzzyAssignments[a][t]
		}
		return
	}, nil
}

// AbsorptionAssignments calculates a majority assignment from absorption probabilities.
func (chain *AbsorbingMarkovChain) AbsorptionAssignments(ctx context.Context) (assigner map[uint32]uint32, err error) {
	fail := func(e error) (map[uint32]uint32, error) {
		assigner, err = nil, e
		return assigner, err
	}

	fuzzyAssignments, ttn, tan, err := chain.absorptionProbabilities(ctx, func() { chain = nil }) //enable eventual GC
	if err != nil {
		return fail(err)
	}

	silentFail := func(fi func(uint32) (uint32, error)) func(int) uint32 {
		return func(intida int) (idb uint32) {
			ida := uint32(intida)
			switch {
			case err != nil:
				//Skip it
			case ida > ^uint32(0): //max Uint32
				err = errors.Errorf("%v is not a valid node.", ida)
			default:
				idb, err = fi(ida)
			}
			return
		}
	}
	ttn2Old := silentFail(ttn.ToOld)
	tan2Old := silentFail(tan.ToOld)

	assigner = make(map[uint32]uint32, len(fuzzyAssignments[0]))
	for tnID := range fuzzyAssignments[0] {
		perm := rand.Perm(len(fuzzyAssignments))
		bestv, bestw := -1, -1.0
		for _, v := range perm {
			w := fuzzyAssignments[v][tnID]
			if w > bestw {
				bestv = v
				bestw = w
			}
		}
		assigner[ttn2Old(tnID)] = tan2Old(bestv)
	}

	if err != nil {
		return fail(err)
	}

	return
}

func (chain *AbsorbingMarkovChain) absorptionProbabilities(ctx context.Context, clean func()) (fuzzyAssignments [][]float64, ttn, tan translator, err error) {
	fail := func(e error) ([][]float64, translator, translator, error) {
		fuzzyAssignments, ttn, tan, err = nil, nil, nil, e
		return fuzzyAssignments, ttn, tan, err
	}

	if err = chain.checkRequirements(); err != nil {
		return fail(err)
	}

	var tmpDir string
	if tmpDir, err = ioutil.TempDir(chain.tmpDir, "."); err != nil {
		return fail(errors.Wrap(err, "AbsorbingMarkovChain Error: unable to create a temporary directory."))
	}
	defer os.RemoveAll(tmpDir)
	solverInfile := filepath.Join(tmpDir, "Ab.ptsc")
	solverOutfile := filepath.Join(tmpDir, "sol.matlab")

	//transform wikigraph to Ab.petsc
	if ttn, tan, err = graph2Petsc(chain, solverInfile); err != nil {
		return fail(err)
	}

	//enable eventual GC
	chain = nil
	clean()
	debug.FreeOSMemory()

	//run solver
	if err = gmres.Run(ctx, solverInfile, solverOutfile, tmpDir); err != nil {
		return fail(err)
	}

	//transform back from sol.matlab
	if fuzzyAssignments, err = petsc2Assignments(ttn, tan, solverOutfile); err != nil {
		return fail(err)
	}

	return
}

func (chain *AbsorbingMarkovChain) checkRequirements() (err error) { //for absorbing markov chain
	if chain == nil {
		return errors.New("AbsorbingMarkovChain Error: nil chain")
	}

	if err = chain.checkGraphNodes(); err == nil {
		return
	}

	nodes := roaring.NewBitmap()

	if err = chain.checkAbsorbingNodes(nodes); err == nil {
		return
	}

	if err = chain.checkTransientNodes(nodes); err == nil {
		return
	}

	if nodes.GetCardinality() != chain.Nodes.GetCardinality() {
		v, _ := roaring.AndNot(chain.Nodes, nodes).Select(0)
		return errors.Errorf("%v isn't transient node, neither it's declared absorbing.", v)
	}

	chain.Weighter = checkedWeighter(chain.Weighter)

	return
}

func (chain *AbsorbingMarkovChain) checkGraphNodes() (err error) {
	nodes := chain.Nodes
	for i := nodes.Iterator(); i.HasNext(); {
		from := i.Next()
		to := chain.Edges(from)
		for _, id := range to {
			if !nodes.Contains(id) {
				return errors.Errorf("arc (%v,%v) shouldn't exist: %v isn't a graph node.", from, id, id)
			}
		}
	}
	return
}

func (chain *AbsorbingMarkovChain) checkAbsorbingNodes(nodes *roaring.Bitmap) (err error) {
	for i := chain.absorbingNodes.Iterator(); i.HasNext(); {
		ANode := i.Next()
		to := chain.Edges(ANode)
		switch {
		case len(to) > 1:
			fallthrough
		case len(to) == 1 && to[0] != ANode:
			return errors.Errorf("%v is not a valid absorbing node.", ANode)
		default:
			nodes.Add(ANode)
		}
	}
	return
}

func (chain *AbsorbingMarkovChain) checkTransientNodes(nodes *roaring.Bitmap) (err error) {
	changed := true
	for changed {
		changed = false
		for i := roaring.AndNot(chain.Nodes, chain.absorbingNodes).Iterator(); i.HasNext(); {
			from := i.Next()
			if !nodes.Contains(from) {
				to := chain.Edges(from)
				for _, id := range to {
					if nodes.Contains(id) {
						nodes.Add(from)
						changed = true
						break
					}
				}
			}
		}
	}
	return
}

func checkedWeighter(weighter func(from, to uint32) (weight float64, err error)) func(from, to uint32) (weight float64, err error) {
	return func(from, to uint32) (weight float64, err error) {
		weight, err = weighter(from, to)
		switch {
		case err != nil:
			//err already set
		case weight <= 0:
			err = errors.Errorf("arc (%v,%v) hasn't positive weight (%v).", from, to, weight)
		case math.IsInf(weight, 0):
			err = errors.Errorf("arc (%v,%v) has infinite weight (%v).", from, to, weight)
		case math.IsNaN(weight):
			err = errors.Errorf("arc (%v,%v) has NaN weight (%v).", from, to, weight)
		}
		return
	}
}
