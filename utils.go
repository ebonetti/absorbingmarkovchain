package absorbingmarkovchain

import (
	//	"fmt"
	"sort"

	"github.com/RoaringBitmap/roaring"
	"github.com/pkg/errors"
)

type translator interface {
	ToNew(oldID uint32) (newID uint32, err error)
	ToOld(newID uint32) (oldID uint32, err error)
}

type myTranslator []uint32

func newTranslator(set *roaring.Bitmap) translator {
	return myTranslator(set.ToArray())
}

func (t myTranslator) ToNew(oldID uint32) (newID uint32, err error) {
	fail := func(e error) (uint32, error) {
		newID, err = 0, e
		return newID, err
	}

	exist, intNewID := uint32Exist(t, oldID)
	if !exist {
		return fail(errors.Errorf("Translator: inexistent oldid %v", oldID))
	}
	newID = uint32(intNewID)
	return
}

func (t myTranslator) ToOld(newID uint32) (oldID uint32, err error) {
	fail := func(e error) (uint32, error) {
		oldID, err = 0, e
		return oldID, err
	}

	if newID >= uint32(len(t)) {
		return fail(errors.Errorf("Translator: inexistent newid %v", newID))
	}

	oldID = t[newID]
	return
}

func uint32Search(a []uint32, x uint32) int {
	return sort.Search(len(a), func(i int) bool { return a[i] >= x })
}

func uint32Exist(a []uint32, x uint32) (exist bool, position int) {
	position = uint32Search(a, x)
	exist = position < len(a) && a[position] == x
	return
}

func fsum(summands []float64) (sum float64) { //Has permission to modify input array
	if len(summands) == 0 {
		return
	}
	//Iterative pairwise summation
	for len(summands) > 1 {
		ndiv2 := len(summands) / 2
		nmod2 := len(summands) % 2
		a := summands[nmod2 : ndiv2+nmod2]
		b := summands[ndiv2+nmod2:]
		for p := range a {
			a[p] += b[p]
		}
		summands = summands[:ndiv2+nmod2]
	}
	sum = summands[0]
	return
}
