//Package gmres provides a wrapper for gmres-petsc.
package gmres

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
)

//go:generate go-bindata -pkg $GOPACKAGE gmres-petsc/...

const solverDir = "gmres-petsc"

//Run executes the gmres command on the given directory with the given context
func Run(ctx context.Context, infile, outfile, tmpdir string) (err error) {
	if err = RestoreAssets(tmpdir, solverDir); err != nil {
		return errors.Wrapf(err, "AbsorbingMarkovChain Error: unable to convert to restore asset %s", solverDir)
	}
	for _, p := range []*string{&infile, &outfile} {
		ap, err := filepath.Abs(*p)
		if err != nil {
			return errors.Wrapf(err, "AbsorbingMarkovChain Error: unable to convert to absolute path %s", *p)
		}
		*p = ap
	}

	cmd := exec.CommandContext(ctx, "make", "run", "IFPATH="+infile, "OFPATH="+outfile)

	var cmdStderr bytes.Buffer
	cmd.Stderr = &cmdStderr
	cmd.Dir = filepath.Join(tmpdir, solverDir)
	defer os.RemoveAll(cmd.Dir)

	//run solver
	if err = cmd.Run(); err != nil {
		return errors.Wrap(err, "AbsorbingMarkovChain Error: call to external command - PETSc GMRES - failed, with the following error stream:\n"+cmdStderr.String())
	}

	return
}
