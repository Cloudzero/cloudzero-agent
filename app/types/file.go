package types

import "io"

type File interface {
	io.ReadWriteCloser

	UniqueID() string          // a unique identifier for the file
	Location() (string, error) // location of the file
	Rename(new string) error   // change the name / location of the file in the environment
}
