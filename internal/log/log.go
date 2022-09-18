package log

import (
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"

	api "github.com/sota0121/proglog/api/v1"
)

const (
	defaultIndexMaxBytes = 1024
	defaultStoreMaxBytes = 1024
)

type Log struct {
	mu sync.RWMutex

	Dir    string
	Config Config

	activeSegment *segment
	segments      []*segment
}

func NewLog(dir string, c Config) (*Log, error) {
	// Check if the config has valid maximum index and store sizes.
	if c.Segment.MaxIndexBytes == 0 {
		c.Segment.MaxIndexBytes = defaultIndexMaxBytes
	}
	if c.Segment.MaxStoreBytes == 0 {
		c.Segment.MaxStoreBytes = defaultStoreMaxBytes
	}

	// Create the log object.
	l := &Log{
		Dir:    dir,
		Config: c,
	}
	return l, l.setup()
}

// setup initializes the log.
// Conditions:
// - the log directory exists
// - the log directory contains only valid segment files
// - the log directory don't necessarily contain any segment files
func (l *Log) setup() error {
	files, err := os.ReadDir(l.Dir)
	if err != nil {
		return err
	}

	// Build segment list from files.
	var baseOffsets []uint64
	for _, file := range files {
		// Get the base offset from the file name. (e.g. 1024.index, 1024.store)
		offStr := strings.TrimSuffix(
			file.Name(),
			path.Ext(file.Name()),
		)
		off, _ := strconv.ParseUint(offStr, 10, 0)
		baseOffsets = append(baseOffsets, off)
	}
	// Sort the base offsets in ascending order.
	sort.Slice(baseOffsets, func(i, j int) bool {
		return baseOffsets[i] < baseOffsets[j]
	})
	// Create a segment for each base offset.
	for i := 0; i < len(baseOffsets); i++ {
		if err := l.newSegment(baseOffsets[i]); err != nil {
			return err
		}
		// baseOffsets contains duplicates for index and store
		// so we skip the next one.
		// (e.g. 0.index, 0.store, 1.index, 1.store ...)
		i++
	}

	// If there are no segments, create a new one.
	if l.segments == nil || len(l.segments) == 0 {
		if err = l.newSegment(
			l.Config.Segment.InitialOffset,
		); err != nil {
			return err
		}
	}
	return nil
}

// newSegment creates a new segment and appends it to the log.
func (l *Log) newSegment(off uint64) error {
	s, err := newSegment(l.Dir, off, l.Config)
	if err != nil {
		return err
	}
	l.segments = append(l.segments, s)
	l.activeSegment = s
	return nil
}

// Append appends a record to the log.
func (l *Log) Append(record *api.Record) (uint64, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// If the active segment is full, create a new one.
	if l.activeSegment.IsMaxed() {
		highestOffset, err := l.highestOffset()
		if err != nil {
			return 0, err
		}

		err = l.newSegment(highestOffset + 1)
		if err != nil {
			return 0, err
		}
	}

	// Append the record to the active segment.
	off, err := l.activeSegment.Append(record)
	if err != nil {
		return 0, err
	}
	return off, nil
}

// highestOffset returns the highest offset in the log.
func (l *Log) highestOffset() (uint64, error) {
	off := l.segments[len(l.segments)-1].nextOffset
	if off == 0 {
		return 0, nil
	}

	return off - 1, nil // highest offset is nextOffset - 1
}

// Read reads a record from the log with the given offset.
func (l *Log) Read(off uint64) (*api.Record, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var s *segment
	for _, seg := range l.segments {
		if seg.baseOffset <= off && off < seg.nextOffset {
			s = seg
			break
		}
	}
	if s == nil {
		return nil, fmt.Errorf("offset out of range: %d", off)
	}
	return s.Read(off)
}

// Close closes the all segments in the log.
func (l *Log) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, seg := range l.segments {
		if err := seg.Close(); err != nil {
			return err
		}
	}
	return nil
}

// Remove removes the log directory and all segment files.
func (l *Log) Remove() error {
	if err := l.Close(); err != nil {
		return err
	}
	return os.RemoveAll(l.Dir)
}

// Reset remove the log and setup a new one.
func (l *Log) Reset() error {
	if err := l.Remove(); err != nil {
		return err
	}
	return l.setup()
}

// LowestOffset returns the lowest base offset of the log.
// Conditions:
// - the log contains at least one segment
func (l *Log) LowestOffset() (uint64, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.segments[0].baseOffset, nil
}

// HighestOffset returns the highest offset of the log.
func (l *Log) HighestOffset() (uint64, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.highestOffset()
}

// Truncate removes the segments which have a next offset less than the given offset.
// This method is used to remove old segments in order to free up disk space.
func (l *Log) Truncate(lowest uint64) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	var segments []*segment
	for _, seg := range l.segments {
		if seg.nextOffset <= lowest+1 {
			if err := seg.Remove(); err != nil {
				return err
			}
		}
		segments = append(segments, seg)
	}
	l.segments = segments // update segments
	return nil
}

// Reader returns a MultiReader which reads all segments in the log.
func (l *Log) Reader() io.Reader {
	l.mu.RLock()
	defer l.mu.RUnlock()

	readers := make([]io.Reader, len(l.segments))
	for i, seg := range l.segments {
		readers[i] = &originReader{seg.store, 0}
	}
	return io.MultiReader(readers...)
}

type originReader struct {
	*store
	off int64
}

func (o *originReader) Read(p []byte) (n int, err error) {
	n, err = o.ReadAt(p, o.off)
	o.off += int64(n)
	return n, err
}
