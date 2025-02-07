package shipper

import (
	"io"
	"os"
)

type FileReader interface {
	Open(path string) (*os.File, error)
	Read(r io.Reader) ([]byte, error)
}

type OSFileReader struct{}

func (OSFileReader) Open(path string) (*os.File, error) {
	return os.Open(path)
}

func (OSFileReader) Read(r io.Reader) ([]byte, error) {
	return io.ReadAll(r)
}

type MockFileReader struct {
	Data    []byte
	OpenErr error
	ReadErr error
}

func NewMockFileReader(data []byte, openErr error, readErr error) *MockFileReader {
	fr := &MockFileReader{
		Data:    data,
		OpenErr: openErr,
		ReadErr: readErr,
	}
	return fr
}

func (fr MockFileReader) Open(path string) (*os.File, error) {
	return nil, fr.OpenErr
}

func (fr MockFileReader) Read(r io.Reader) ([]byte, error) {
	return fr.Data, fr.ReadErr
}
