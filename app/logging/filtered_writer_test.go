package logging_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/cloudzero/cloudzero-agent/app/logging"
)

func TestUnit_Logging_FilteredWriter_EmptyWrite(t *testing.T) {
	buf := &bytes.Buffer{}
	w := logging.NewFieldFilterWriter(buf, []string{"foo"})
	n, err := w.Write([]byte{})
	if wantN, wantErr := 0, error(nil); n != wantN || err != wantErr {
		t.Fatalf("Write(empty) = (%d, %v), want (%d, %v)", n, err, wantN, wantErr)
	}
	if buf.Len() != 0 {
		t.Fatalf("underlying buffer should be empty, got %q", buf.String())
	}
}

func TestUnit_Logging_FilteredWriter_FilterRemovesFields(t *testing.T) {
	var buf bytes.Buffer
	orig := `{"keep":"yes","skip":"no"}`
	w := logging.NewFieldFilterWriter(&buf, []string{"skip"})
	n, err := w.Write([]byte(orig))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != len(orig) {
		t.Fatalf("n = %d, want %d", n, len(orig))
	}
	out := buf.String()
	// after filtering only "keep" should remain
	const want = `{"keep":"yes"}`
	if out != want {
		t.Fatalf("filtered = %q, want %q", out, want)
	}
}

func TestUnit_Logging_FilteredWriter_PreserveNewline(t *testing.T) {
	var buf bytes.Buffer
	orig := `{"a":1}` + "\n"
	w := logging.NewFieldFilterWriter(&buf, nil)
	n, err := w.Write([]byte(orig))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != len(orig) {
		t.Fatalf("n = %d, want %d", n, len(orig))
	}
	out := buf.String()
	const want = `{"a":1}` + "\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestUnit_Logging_FilteredWriter_MalformedJSONWritesAsIs(t *testing.T) {
	var buf bytes.Buffer
	bad := `not a json`
	w := logging.NewFieldFilterWriter(&buf, []string{"x"})
	n, err := w.Write([]byte(bad))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != len(bad) {
		t.Fatalf("n = %d, want %d", n, len(bad))
	}
	if buf.String() != bad {
		t.Fatalf("buf = %q, want %q", buf.String(), bad)
	}
}

func TestUnit_Logging_FilteredWriter_UnderlyingWriteError(t *testing.T) {
	testErr := errors.New("boom")
	ew := &errorWriter{err: testErr}
	w := logging.NewFieldFilterWriter(ew, []string{"foo"})
	data := []byte(`{"foo":"x"}`)
	n, err := w.Write(data)
	// since we filtered out "foo", marshal succeeds, but Write returns error
	if err != testErr {
		t.Fatalf("err = %v, want %v", err, testErr)
	}
	// errorWriter always returns written=0
	if n != 0 {
		t.Fatalf("n = %d, want 0", n)
	}
}

func TestUnit_Logging_FilteredWriter_PartialWriteIgnored(t *testing.T) {
	pw := &partialWriter{n: 0} // writes 0 bytes, no error
	w := logging.NewFieldFilterWriter(pw, []string{"foo"})
	data := []byte(`{"foo":"x","bar":"y"}`)
	n, err := w.Write(data)
	// no error, and we always return original length
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if n != len(data) {
		t.Fatalf("n = %d, want %d", n, len(data))
	}
}

// errorWriter always returns (0, err)
type errorWriter struct{ err error }

func (e *errorWriter) Write(p []byte) (int, error) {
	return 0, e.err
}

// partialWriter returns a short write with no error
type partialWriter struct{ n int }

func (p *partialWriter) Write(b []byte) (int, error) {
	return p.n, nil
}
