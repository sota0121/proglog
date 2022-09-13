package log

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	write       = []byte("hello world")         // the data to write
	width       = uint64(len(write)) + lenWidth // the length of the record
	numOfRecord = uint64(3)                     // the number of records
)

func TestStoreAppendRead(t *testing.T) {
	// Create a temporary file.
	f, err := os.CreateTemp("", "store_append_read_test")
	require.NoError(t, err)
	defer os.Remove(f.Name())

	// Create a new store.
	s, err := newStore(f)
	require.NoError(t, err)

	testAppend(t, s)
	testRead(t, s)
	testReadAt(t, s)

	// test if the service can be excuted again after restart
	s, err = newStore(f)
	require.NoError(t, err)
	testRead(t, s)

}

func testAppend(t *testing.T, s *store) {
	t.Helper() // mark this function as a helper function

	for i := uint64(1); i < (numOfRecord + 1); i++ {
		n, pos, err := s.Append(write)
		require.NoError(t, err)

		// check the index of the record in 2 ways
		require.Equal(t, pos+n, width*i)
	}
}

func testRead(t *testing.T, s *store) {
	t.Helper() // mark this function as a helper function

	var pos uint64
	for i := uint64(1); i < (numOfRecord + 1); i++ {
		read, err := s.Read(pos)
		require.NoError(t, err)
		require.Equal(t, write, read)
		pos += width
	}
}

func testReadAt(t *testing.T, s *store) {
	t.Helper() // mark this function as a helper function

	for i, off := uint64(1), int64(0); i < (numOfRecord + 1); i++ {
		b := make([]byte, lenWidth)
		n, err := s.ReadAt(b, off) // read the length of the record
		require.NoError(t, err)
		require.Equal(t, lenWidth, n)
		off += int64(n)

		size := enc.Uint64(b)
		b = make([]byte, size)
		n, err = s.ReadAt(b, off) // read the record
		require.NoError(t, err)
		require.Equal(t, write, b)
		require.Equal(t, size, uint64(n))
		off += int64(n)
	}
}

func TestStoreClose(t *testing.T) {
	// Create a temporary file.
	f, err := os.CreateTemp("", "store_close_test")
	require.NoError(t, err)
	defer os.Remove(f.Name())

	// Write some data to the temporary file.
	// but actually the data is written to the buffer not the file.
	s, err := newStore(f)
	require.NoError(t, err)
	_, _, err = s.Append(write)
	require.NoError(t, err)

	// Get the file size before closing the store.
	f, beforeSize, err := openFile(f.Name())
	require.NoError(t, err)

	// Act
	err = s.Close()
	require.NoError(t, err)

	// Get the file size after closing the store.
	_, afterSize, err := openFile(f.Name())
	require.NoError(t, err)
	require.True(t, afterSize > beforeSize)

}

func openFile(name string) (file *os.File, size int64, err error) {

	f, err := os.OpenFile(
		name,
		os.O_RDWR|os.O_CREATE|os.O_APPEND,
		0644,
	)
	if err != nil {
		return nil, 0, err
	}
	fi, err := f.Stat()
	if err != nil {
		return nil, 0, err
	}
	return f, fi.Size(), nil
}
