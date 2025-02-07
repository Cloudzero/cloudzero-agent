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

type MockFileReaderOpt = func(fr *MockFileReader)

func WithMockFileReaderData(data []byte) MockFileReaderOpt {
	return func(fr *MockFileReader) {
		fr.Data = data
	}
}

func WithMockFileReaderFileOpenError(err error) MockFileReaderOpt {
	return func(fr *MockFileReader) {
		fr.OpenErr = err
	}
}

func WithMockFileReaderFileReadError(err error) MockFileReaderOpt {
	return func(fr *MockFileReader) {
		fr.ReadErr = err
	}
}

func NewMockFileReader(opts ...MockFileReaderOpt) *MockFileReader {
	fr := &MockFileReader{}
	for _, opt := range opts {
		opt(fr)
	}
	return fr
}

func (fr MockFileReader) Open(path string) (*os.File, error) {
	return nil, fr.OpenErr
}

func (fr MockFileReader) Read(r io.Reader) ([]byte, error) {
	return fr.Data, fr.ReadErr
}
