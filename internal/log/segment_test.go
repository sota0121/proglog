package log

import (
	"io"
	"os"
	"testing"

	api "github.com/sota0121/proglog/api/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// TestSegment tests the segment.
func TestSegment(t *testing.T) {
	dir, err := os.MkdirTemp("", "segment_test")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	// Create a test target record.
	want := &api.Record{Value: []byte("hello world")}

	// ======== Test case 1 ========
	// Create a new segment so that the index is maxed but the store is not.
	numOfEntries := uint64(3)
	c := Config{}
	c.Segment.MaxStoreBytes = 1024
	c.Segment.MaxIndexBytes = entWidth * numOfEntries

	baseOffset := uint64(16)
	s, err := newSegment(dir, baseOffset, c)
	require.NoError(t, err)
	require.Equal(t, baseOffset, s.nextOffset)
	require.False(t, s.IsMaxed())

	// Append the record to the segment until it is maxed.
	for i := uint64(0); i < numOfEntries; i++ {
		off, err := s.Append(want)
		require.NoError(t, err)
		require.Equal(t, baseOffset+i, off)

		got, err := s.Read(off)
		require.NoError(t, err)
		require.Equal(t, want.Value, got.Value)
	}

	// Additional append should fail because the segment is maxed.
	_, err = s.Append(want)
	require.Equal(t, io.EOF, err)

	// The segment should be maxed because the index is maxed.
	require.True(t, s.IsMaxed())
	require.NoError(t, s.Close())

	// ======== Test case 2 ========
	// Restructure the segment so that store is maxed but index is not.
	p, _ := proto.Marshal(want)
	c.Segment.MaxStoreBytes = uint64(len(p)+lenWidth) * (numOfEntries + 1)
	c.Segment.MaxIndexBytes = 1024

	s, err = newSegment(dir, baseOffset, c)
	require.NoError(t, err)

	// The segment should be maxed because the store is maxed.
	require.True(t, s.IsMaxed())

	// Remove the segment.
	require.NoError(t, s.Remove())

	// ======== Test case 3 ========
	// Create a new segment, and append no record.
	s, err = newSegment(dir, baseOffset, c)
	require.NoError(t, err)
	require.False(t, s.IsMaxed())
	require.NoError(t, s.Close())
}
