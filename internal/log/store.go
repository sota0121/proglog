package log

import (
	"bufio"
	"encoding/binary"
	"os"
	"sync"
)

var (
	enc = binary.BigEndian
)

const (
	lenWidth = 8 // bytes
)

type store struct {
	*os.File
	mu   sync.Mutex
	buf  *bufio.Writer
	size uint64
}

func newStore(f *os.File) (*store, error) {
	fi, err := os.Stat(f.Name())
	if err != nil {
		return nil, err
	}

	size := uint64(fi.Size())
	return &store{
		File: f,
		size: size,
		buf:  bufio.NewWriter(f),
	}, nil
}

// Append appends a record to the store.
// this method returns the offset of the record.
// n: the length of the buffer
// pos: the offset of the record
func (s *store) Append(p []byte) (n uint64, pos uint64, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pos = s.size

	// Write the length of the record.
	if err := binary.Write(s.buf, enc, uint64(len(p))); err != nil {
		return 0, 0, err
	}

	// Write the record.
	w, err := s.buf.Write(p)
	if err != nil {
		return 0, 0, err
	}

	// the size of record is record + meta (len of record)
	w += lenWidth

	s.size += uint64(w)
	return uint64(w), pos, nil
}

// Read reads a record from the store with the given offset.
func (s *store) Read(pos uint64) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Flush the buffer before reading.
	// Flush means that all data in the buffer is written to the file.
	if err := s.buf.Flush(); err != nil {
		return nil, err
	}

	// Read the length of the record.
	size := make([]byte, lenWidth)
	if _, err := s.File.ReadAt(size, int64(pos)); err != nil {
		return nil, err
	}

	// Read the record using the record size from the previous step.
	b := make([]byte, enc.Uint64(size))
	if _, err := s.File.ReadAt(b, int64(pos+lenWidth)); err != nil {
		return nil, err
	}
	return b, nil
}

// ReadAt is the wrapper of ReadAt method of os.File.
// This is the interface of io.ReaderAt.
func (s *store) ReadAt(p []byte, off int64) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Flush the buffer before reading.
	// Flush means that all data in the buffer is written to the file.
	if err := s.buf.Flush(); err != nil {
		return 0, err
	}

	// Invoke ReadAt method of os.File.
	return s.File.ReadAt(p, off)
}

// Close closes the store.
func (s *store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Flush the buffer before closing.
	// Flush means that all data in the buffer is written to the file.
	err := s.buf.Flush()
	if err != nil {
		return err
	}
	return s.File.Close()
}
