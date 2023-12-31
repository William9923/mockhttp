package mockhttp

import (
	"bytes"
	"io"
	"reflect"
	"testing"
)

func TestReusableReader_Read(t *testing.T) {
	data := []byte("Hello, world!")
	reader := bytes.NewReader(data)
	reusable := ReusableReader(reader).(reusableReader)

	// Read the whole content
	buffer := make([]byte, len(data))
	n, err := reusable.Read(buffer)
	if err != nil && err != io.EOF {
		t.Errorf("Error reading: %s", err)
	}
	if n != len(data) {
		t.Errorf("Read %d bytes, expected %d", n, len(data))
	}
	if !reflect.DeepEqual(buffer[:n], data) {
		t.Errorf("Data mismatch")
	}

	// Reset and read again
	reusable.reset() // automatically closed during runtime by io.ReadCloser
	n, err = reusable.Read(buffer)
	if err != nil && err != io.EOF {
		t.Errorf("Error reading after reset: %s", err)
	}
	if n != len(data) {
		t.Errorf("Read %d bytes after reset, expected %d", n, len(data))
	}
	if !reflect.DeepEqual(buffer[:n], data) {
		t.Errorf("Data mismatch after reset")
	}
}
