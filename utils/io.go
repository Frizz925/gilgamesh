package utils

import "io"

func WriteFull(w io.Writer, b []byte) error {
	for n := 0; n < len(b); {
		nn, err := w.Write(b[n:])
		if err != nil {
			return err
		}
		n += nn
	}
	return nil
}
