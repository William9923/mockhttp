package mockhttp

import (
	"bytes"
	"io"
)

// reusableReader is a custom type implementing the io.Reader interface, enhancing it with
// the ability to reset and re-read the underlying data efficiently.
type reusableReader struct {
	io.Reader
	readBuf *bytes.Buffer
	backBuf *bytes.Buffer
}

// ReusableReader creates and returns a new reusableReader based on the provided io.Reader.
// The reusableReader allows for multiple reads of the same data efficiently.
func ReusableReader(r io.Reader) io.Reader {
	readBuf := bytes.Buffer{}
	readBuf.ReadFrom(r) // error handling ignored for brevity
	backBuf := bytes.Buffer{}

	return reusableReader{
		io.TeeReader(&readBuf, &backBuf),
		&readBuf,
		&backBuf,
	}
}

// Read reads data into the provided byte slice and returns the number of bytes read.
// If the end of the underlying data is reached (io.EOF), it automatically resets the reader for
// subsequent reads.
func (r reusableReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	if err == io.EOF {
		r.reset()
	}
	return n, err
}

// reset rewinds the reusableReader by copying the captured data from the backup buffer
// back to the main read buffer, allowing for the underlying data to be read again.
func (r reusableReader) reset() {
	io.Copy(r.readBuf, r.backBuf) // nolint: errcheck
}
