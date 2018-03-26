package absorbingmarkovchain

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"os"
	"sort"

	"github.com/RoaringBitmap/roaring"
	"github.com/pkg/errors"
)

func (g dGraph) Print() {
	for i := g.Nodes.Iterator(); i.HasNext(); {
		from := i.Next()
		fmt.Println(from, g.Edges(from))
	}
}

func (g wDGraph) Print() {
	for i := g.Nodes.Iterator(); i.HasNext(); {
		from := i.Next()
		to := g.Edges(from)
		ww := []float64{}
		for _, to := range to {
			if w, err := g.Weighter(from, to); err != nil {
				panic(err)
			} else {
				ww = append(ww, w)
			}
		}
		fmt.Println(from, to, ww)
	}
}

func graph2Petsc(chain *AbsorbingMarkovChain, filepath string) (ttn, tan translator, err error) {
	fail := func(e error) (translator, translator, error) {
		ttn, tan, err = nil, nil, e
		return ttn, tan, err
	}

	Ab, err := os.Create(filepath)
	if err != nil {
		return fail(errors.Wrapf(err, "AbsorbingMarkovChain Error: unable to create a temporary file at %v.", filepath))
	}
	defer func() {
		if e := Ab.Close(); e != nil && err == nil {
			/*return*/ fail(e)
		}
	}()

	w := bufio.NewWriter(Ab)
	defer func() {
		if e := w.Flush(); e != nil && err == nil {
			/*return*/ fail(e)
		}
	}()

	write := func(vv ...interface{}) {
		for _, v := range vv {
			if err != nil {
				return
			}
			//_,err = fmt.Fprintln(w,v)
			err = binary.Write(w, binary.BigEndian, v)
		}

	}

	ttn, tan, e := _graph2Petsc(chain, write)
	if err != nil {
		return fail(errors.Wrapf(err, "AbsorbingMarkovChain Error: error while writing file at %v.", filepath))
	}
	err = e

	return
}

const matFileClassID int32 = 1211216
const vecFileClassID int32 = 1211214

func _graph2Petsc(chain *AbsorbingMarkovChain, write func(...interface{})) (ttn, tan translator, err error) {
	fail := func(e error) (t1, t2 translator, err error) {
		ttn, tan, err = nil, nil, e
		return ttn, tan, err
	}
	g := chain.FilterNodes(chain.absorbingNodes).AddSelfLoops()
	n := uint32(g.Nodes.GetCardinality())
	entries, rowEntries := uint32(0), make([]uint32, 0, n)
	for i := g.Nodes.Iterator(); i.HasNext(); {
		from := i.Next()
		l := uint32(len(g.Edges(from)))
		entries += l
		rowEntries = append(rowEntries, l)
	}
	g, ttn = g.NormalizedIDs()

	wg, err := chain.NormalizedWeights()
	if err != nil {
		return fail(err)
	}
	cb, err := compressedB(&AbsorbingMarkovChain{wg, chain.absorbingNodes})
	if err != nil {
		return fail(err)
	}
	wg = wg.AddSelfLoops()
	wg.dGraph = wg.dGraph.FilterNodes(chain.absorbingNodes)

	/*
			MAT_FILE_CLASSID //matrix file identifier
			n               //number of rows
			n,         //number of columns
		    entries,   //total number of nonzeros
			rowEntries,//number nonzeros in each row
			indices,   //column indices of all nonzeros
			values,    //values of all nonzeros
	*/
	write(matFileClassID, n, n, entries, rowEntries)
	for i := g.Nodes.Iterator(); i.HasNext(); {
		from := i.Next()
		if to := g.Edges(from); len(to) > 0 {
			write(to)
		}
	}
	for i := wg.Nodes.Iterator(); i.HasNext(); {
		from := i.Next()
		for _, to := range wg.Edges(from) {
			w, err := wg.Weighter(from, to)
			if err != nil {
				return fail(err)
			}
			write(w)
		}
	}

	b := make([]float64, 0, n)
	for i := chain.absorbingNodes.Iterator(); i.HasNext(); {
		b = b[:0]
		for _, e := range cb[i.Next()] {
			p, err := ttn.ToNew(e.to)
			if err != nil {
				return fail(err)
			}
			for uint32(len(b)) < p {
				b = append(b, 0)
			}
			b = append(b, e.w)
		}
		for uint32(len(b)) < n {
			b = append(b, 0)
		}

		/*
		   VEC_FILE_CLASSID, //vector file identifier
		   n,         //number of rows
		   b,    //values of all entries
		*/
		write(vecFileClassID, n)
		write(b)
	}

	tan = newTranslator(chain.absorbingNodes)

	return
}

type implicitWeightedEdge struct {
	to uint32
	w  float64
}

func compressedB(chain *AbsorbingMarkovChain) (cb map[uint32][]implicitWeightedEdge, err error) {
	cb = map[uint32][]implicitWeightedEdge{}
	for i := roaring.AndNot(chain.Nodes, chain.absorbingNodes).Iterator(); i.HasNext(); {
		from := i.Next()
		to := roaring.BitmapOf(chain.Edges(from)...)
		to.And(chain.absorbingNodes)
		for i := to.Iterator(); i.HasNext(); {
			to := i.Next()
			tt := cb[to]
			p := sort.Search(len(tt), func(i int) bool { return tt[i].to >= from })
			tt = append(tt, implicitWeightedEdge{})
			copy(tt[p+1:], tt[p:])
			e := implicitWeightedEdge{to: from}
			if e.w, err = chain.Weighter(from, to); err != nil {
				return
			}
			e.w = -e.w
			tt[p] = e
			cb[to] = tt
		}
	}
	return
}
