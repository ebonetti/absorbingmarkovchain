package absorbingmarkovchain

import (
	"bufio"
	"io"

	"github.com/pkg/errors"
)

func newMatlab2Json(file io.Reader) io.Reader {
	return &matlab2Json{file: bufio.NewReader(file)}
}

type matlab2Json struct {
	file   *bufio.Reader
	buffer []byte
	err    error
}

func (r *matlab2Json) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	//len(p)>0

	if len(r.buffer) == 0 && r.refill() != nil {
		return 0, r.err
	}
	//len(r.buffer)>0

	min := len(p)
	if len(r.buffer) < len(p) {
		min = len(r.buffer)
	}
	//m>0

	copy(p, r.buffer[:min])
	n, r.buffer = min, r.buffer[min:]

	return n, nil
}

func (r *matlab2Json) refill() (err error) {
	if r.err != nil {
		return r.err
	}

	defer func() {
		r.err = err
	}()

	_, err = r.file.ReadBytes('[')
	switch err {
	case nil:
		//do nothing
	case io.EOF:
		return err
	default:
		return errors.Wrap(err, "AbsorbingMarkovChain Error: error while reading from reader")
	}

	buffer, err := r.file.ReadBytes(']')
	blen := len(buffer)
	switch {
	case blen != 0:
		//do nothing
	case err == io.EOF:
		return err
	default:
		return errors.Wrap(err, "AbsorbingMarkovChain Error: error while reading from reader")
	}

	//ignore eventual errors if len(buffer)>0

	if buffer[0] != '\n' || buffer[blen-2] != '\n' {
		from := blen - 100
		if from < 0 {
			from = 0
		}

		message := "AbsorbingMarkovChain Error: invalid input, ends with ...'" + string(buffer[from:]) + "'."
		if err != nil {
			return errors.Wrap(err, message)
		}
		return errors.Errorf(message)
	}

	buffer[0] = '['
	buffer[blen-2] = ']'
	buffer[blen-1] = '\n'

	for i, c := range buffer[:len(buffer)-2] {
		if c == '\n' {
			buffer[i] = ','
		}
	}
	r.buffer = buffer
	return nil
}

func (r *matlab2Json) UnreadByte() error {
	return errors.Errorf("AbsorbingMarkovChain Error: UnreadByte() not implemented.")
}
