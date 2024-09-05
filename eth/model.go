package eth

import "io"

type EthContext struct {
	Name string
	io.ReadWriteCloser
}
