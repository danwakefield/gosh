package T

import "io"

type IOContainer struct {
	In  io.Reader
	Out io.Writer
	Err io.Writer
}
