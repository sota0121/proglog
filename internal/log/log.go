package log

import (
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
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
