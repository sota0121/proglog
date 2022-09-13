package log

import (
	"io"
	"os"

	"github.com/tysonmote/gommap"
)

const (
	offWidth uint64 = 4                   // offset width in bytes
	posWidth uint64 = 8                   // position width in bytes
	entWidth        = offWidth + posWidth // entry width in bytes
)

// Index is a memory-mapped index of a log.
type index struct {
	file *os.File
	mmap gommap.MMap
	size uint64
}

// newIndex creates a new index for the given file.
func newIndex(f *os.File, c Config) (*index, error) {
	// Create the index file.
	idx := &index{
		file: f,
	}

	// Set the current size of the file.
	fi, err := os.Stat(f.Name())
	if err != nil {
		return nil, err
	}
	idx.size = uint64(fi.Size())

	// Change the size of the file to match the configuration.
	if err = os.Truncate(f.Name(), int64(c.Segment.MaxIndexBytes)); err != nil {
		return nil, err
	}

	// Create a new mapping in the virtual address space of the process.
	if idx.mmap, err = gommap.Map(idx.file.Fd(), gommap.PROT_READ|gommap.PROT_WRITE, gommap.MAP_SHARED); err != nil {
		return nil, err
	}
	return idx, nil
}

// Close closes the index in the way of graceful shutdown.
func (i *index) Close() error {
	// Persist the memory-mapped file.
	if err := i.mmap.Sync(gommap.MS_SYNC); err != nil {
		return err
	}
	// Persist the file data to disk.
	if err := i.file.Sync(); err != nil {
		return err
	}
	// Shrinks the file to the size of the data written.
	if err := i.file.Truncate(int64(i.size)); err != nil {
		return err
	}
	return i.file.Close()
}

// Read returns the position of the record with the given offset.
func (i *index) Read(in int64) (out uint32, pos uint64, err error) {
	// Read the offset.
	// We use uint32 to store the offset from the beginning of the segment.
	// Using relative offset instead of absolute offset can reduce the size of the index
	// from 8 bytes to 4 bytes.
	if i.size == 0 {
		return 0, 0, io.EOF
	}
	if in == -1 {
		out = uint32((i.size / entWidth) - 1)
	} else {
		out = uint32(in)
	}
	pos = uint64(out) * entWidth
	if i.size < pos+entWidth {
		return 0, 0, io.EOF
	}
	out = enc.Uint32(i.mmap[pos : pos+offWidth])
	pos = enc.Uint64(i.mmap[pos+offWidth : pos+entWidth])
	return out, pos, nil
}

// Write writes the offset and position of a record to the index.
func (i *index) Write(off uint32, pos uint64) error {
	// Check if the index contains enough space to write the entry.
	if i.isMaxed() {
		return io.EOF
	}

	// Write the offset.
	enc.PutUint32(i.mmap[i.size:i.size+offWidth], off)
	// Write the position.
	enc.PutUint64(i.mmap[i.size+offWidth:i.size+entWidth], pos)

	// Increment the size of the index to the next entry.
	i.size += uint64(entWidth)
	return nil
}

func (i *index) isMaxed() bool {
	return uint64(len(i.mmap)) < i.size+entWidth
}

func (i *index) Name() string {
	return i.file.Name()
}
