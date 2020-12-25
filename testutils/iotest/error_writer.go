package iotest

import "io"

type errorWriter struct {
	err error
}

func NewErrorWriter(err error) io.Writer {
	return &errorWriter{
		err: err,
	}
}

func (ew *errorWriter) Write(_ []byte) (int, error) {
	return 0, ew.err
}
