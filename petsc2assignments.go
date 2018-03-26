package absorbingmarkovchain

import (
	"encoding/json"
	"io"
	"os"

	"github.com/pkg/errors"
)

func petsc2Assignments(ttn, tan translator, filepath string) (fuzzyAssignments [][]float64, err error) {
	fail := func(e error) ([][]float64, error) {
		fuzzyAssignments, err = nil, e
		return fuzzyAssignments, err
	}

	solutions, err := os.Open(filepath)
	if err != nil {
		return fail(errors.Wrapf(err, "AbsorbingMarkovChain Error: error while opening file at %v.", filepath))
	}
	defer solutions.Close()

	d := json.NewDecoder(newMatlab2Json(solutions))
	fa := []float64{}
loop:
	for {
		fa = fa[:0]
		err = d.Decode(&fa)
		switch err {
		case nil:
			fuzzyAssignments = append(fuzzyAssignments, append([]float64{}, fa...))
		case io.EOF:
			err = nil
			break loop
		default:
			return fail(errors.Wrapf(err, "AbsorbingMarkovChain Error: error while decoding file at %v.", filepath))
		}
	}

	return
}
