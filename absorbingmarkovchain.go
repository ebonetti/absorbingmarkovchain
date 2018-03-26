// Package absorbingmarkovchain provides defines for computing absorption probabilities of absorbing markov chains.
package absorbingmarkovchain

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"

	"github.com/RoaringBitmap/roaring"
	"github.com/pkg/errors"
)

// TODO: extend ctx support

// New creates a new absorbing markov chain.
func New(nodes, absorbingNodes *roaring.Bitmap, edges func(from uint32) (to []uint32), weighter func(from, to uint32) (weight float64, err error)) *AbsorbingMarkovChain {
	return &AbsorbingMarkovChain{
		wDGraph{
			dGraph{
				nodes,
				edges,
			},
			weighter,
		},
		absorbingNodes,
	}
}

// AbsorbingMarkovChain represents an absorbing markov chain.
type AbsorbingMarkovChain struct {
	wDGraph
	absorbingNodes *roaring.Bitmap
}

//go:generate go-bindata -pkg $GOPACKAGE petsc-gmres/...

const (
	solverDir     = "petsc-gmres"
	solverInfile  = "Ab.ptsc"
	solverOutfile = "sol.matlab"
)

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
func (chain *AbsorbingMarkovChain) AbsorptionAssignments(ctx context.Context) (assigner func(from uint32) (to uint32, ok bool), err error) {
	fail := func(e error) (func(from uint32) (to uint32, ok bool), error) {
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

	assignment := make(map[uint32]uint32, len(fuzzyAssignments[0]))
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
		assignment[ttn2Old(tnID)] = tan2Old(bestv)
	}

	if err != nil {
		return fail(err)
	}

	return func(from uint32) (to uint32, ok bool) {
		to, ok = assignment[from]
		return
	}, nil
}

func (chain *AbsorbingMarkovChain) absorptionProbabilities(ctx context.Context, clean func()) (fuzzyAssignments [][]float64, ttn, tan translator, err error) {
	fail := func(e error) ([][]float64, translator, translator, error) {
		fuzzyAssignments, ttn, tan, err = nil, nil, nil, e
		return fuzzyAssignments, ttn, tan, err
	}

	if err = chain.checkRequirements(); err != nil {
		return fail(err)
	}

	var dir string
	if dir, err = os.Getwd(); err != nil {
		return fail(errors.Wrap(err, "AbsorbingMarkovChain Error: error while unable to locate current working directory."))
	}
	if dir, err = ioutil.TempDir(dir, "."); err != nil {
		return fail(errors.Wrap(err, "AbsorbingMarkovChain Error: unable to create a temporary directory."))
	}
	defer os.RemoveAll(dir)

	if err = RestoreAssets(dir, solverDir); err != nil {
		return fail(err)
	}

	cmd := exec.CommandContext(ctx,
		"make",
		"run",
		"IFPATH="+solverInfile,
		"OFPATH="+solverOutfile,
		fmt.Sprint("IMAX=", chain.absorbingNodes.GetCardinality()))

	var cmdStderr bytes.Buffer
	cmd.Stderr = &cmdStderr
	cmd.Dir = filepath.Join(dir, solverDir)
	defer os.RemoveAll(cmd.Dir)

	//transform wikigraph to Ab.petsc
	if ttn, tan, err = graph2Petsc(chain, filepath.Join(cmd.Dir, solverInfile)); err != nil {
		return fail(err)
	}

	//enable eventual GC
	chain = nil
	clean()
	debug.FreeOSMemory()

	//run solver
	if err = cmd.Run(); err != nil {
		return fail(errors.Wrap(err, "AbsorbingMarkovChain Error: call to external command - PETSc GMRES - failed, with the following error stream:\n"+cmdStderr.String()))
	}

	//transform back from sol.matlab
	if fuzzyAssignments, err = petsc2Assignments(ttn, tan, filepath.Join(cmd.Dir, solverOutfile)); err != nil {
		return fail(err)
	}

	return
}

func (chain *AbsorbingMarkovChain) checkRequirements() (err error) { //for absorbing markov chain
	fail := func(e error) error {
		err = e
		return err
	}

	if chain == nil {
		return errors.New("AbsorbingMarkovChain Error: nil chain")
	}

	nodes := chain.Nodes
	for i := nodes.Iterator(); i.HasNext(); {
		from := i.Next()
		to := chain.Edges(from)
		for _, id := range to {
			if !nodes.Contains(id) {
				return fail(errors.Errorf("arc (%v,%v) shouldn't exist: %v isn't a graph node.", from, id, id))
			}
		}
	}

	whitelist := roaring.NewBitmap()
	for i := chain.absorbingNodes.Iterator(); i.HasNext(); {
		ANode := i.Next()
		to := chain.Edges(ANode)
		switch {
		case len(to) > 1:
			fallthrough
		case len(to) == 1 && to[0] != ANode:
			return fail(errors.Errorf("%v is not a valid absorbing node.", ANode))
		default:
			whitelist.Add(ANode)
		}
	}

	changed := true
	for changed {
		changed = false
		for i := roaring.AndNot(chain.Nodes, chain.absorbingNodes).Iterator(); i.HasNext(); {
			from := i.Next()
			if !whitelist.Contains(from) {
				to := chain.Edges(from)
				for _, id := range to {
					if whitelist.Contains(id) {
						whitelist.Add(from)
						changed = true
						break
					}
				}
			}
		}
	}
	if whitelist.GetCardinality() != nodes.GetCardinality() {
		v, _ := roaring.AndNot(nodes, whitelist).Select(0)
		return fail(errors.Errorf("%v isn't transient node, neither it's declared absorbing.", v))
	}

	chain.Weighter = checkedWeighter(chain.Weighter)

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
