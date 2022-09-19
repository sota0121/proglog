package log

import (
	"io"
	"os"
	"testing"

	api "github.com/sota0121/proglog/api/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestLog(t *testing.T) {
	testMap := map[string]func(t *testing.T, log *Log){
		"append and read a record succeeds": testAppendRead,
		"offset out of range error":         testOutOfRangeErr,
		"init with existing segments":       testInitExisting,
		"reader":                            testReader,
		"truncate":                          testTruncate,
	}

	for scenario, fn := range testMap {
		t.Run(scenario, func(t *testing.T) {
			dir, err := os.MkdirTemp("", "log-test")
			require.NoError(t, err)
			defer os.RemoveAll(dir)

			c := Config{}
			c.Segment.MaxStoreBytes = 32
			log, err := NewLog(dir, c)
			require.NoError(t, err)

			fn(t, log)
		})
	}
}

func testAppendRead(t *testing.T, log *Log) {
	append := &api.Record{
		Value: []byte("hello world"),
	}
	off, err := log.Append(append)
	require.NoError(t, err)
	require.Equal(t, uint64(0), off) // 0 is the offset of the first record

	read, err := log.Read(off)
	require.NoError(t, err)
	require.Equal(t, append.Value, read.Value) // the value of the record should be the same
	require.NoError(t, log.Close())            // log should be able to close successfully
}

func testOutOfRangeErr(t *testing.T, log *Log) {
	read, err := log.Read(100)                   // 100 is an offset that does not exist
	require.Nil(t, read)                         // the record should be nil
	require.Error(t, err)                        // an error should be returned
	apiErr := err.(api.ErrOffsetOutOfRange)      // get the error as an api error
	require.Equal(t, uint64(100), apiErr.Offset) // the offset of the error should be 100
	require.NoError(t, log.Close())              // log should be able to close successfully
}

func testInitExisting(t *testing.T, o *Log) {
	// Build a log with some records.
	append := &api.Record{
		Value: []byte("hello world"),
	}
	numOfEntries := int64(3)
	for i := 0; i < int(numOfEntries); i++ {
		_, err := o.Append(append)
		require.NoError(t, err)
	}
	require.NoError(t, o.Close())

	// Check if the log range is correct.
	lowOff, err := o.LowestOffset()
	require.NoError(t, err)
	require.Equal(t, uint64(0), lowOff)
	highOff, err := o.HighestOffset()
	require.NoError(t, err)
	require.Equal(t, uint64(numOfEntries-1), highOff)

	// Create a new log while the old log is still open.
	n, err := NewLog(o.Dir, o.Config) // n is a new log
	require.NoError(t, err)

	// Check if the new log has the same range as the old log.
	lowOff, err = n.LowestOffset()
	require.NoError(t, err)
	require.Equal(t, uint64(0), lowOff)
	highOff, err = n.HighestOffset()
	require.NoError(t, err)
	require.Equal(t, uint64(numOfEntries-1), highOff)
	require.NoError(t, n.Close()) // the new log should be able to close successfully
}

// testReader tests if the log can be read what it is written in disk.
func testReader(t *testing.T, log *Log) {
	append := &api.Record{
		Value: []byte("hello world"),
	}
	off, err := log.Append(append)
	require.NoError(t, err)
	require.Equal(t, uint64(0), off)

	reader := log.Reader()
	b, err := io.ReadAll(reader) // read all the records from the log
	require.NoError(t, err)

	read := &api.Record{}
	err = proto.Unmarshal(b[lenWidth:], read) // unmarshal the record
	require.NoError(t, err)
	require.Equal(t, append.Value, read.Value)
	require.NoError(t, log.Close())
}

// testReader tests it can truncate the old segments from the log.
func testTruncate(t *testing.T, log *Log) {
	append := &api.Record{
		Value: []byte("hello world"),
	}
	numOfEntries := 3
	for i := 0; i < numOfEntries; i++ {
		_, err := log.Append(append)
		require.NoError(t, err)
	}

	// Truncate the log to the 1 oldest record.
	err := log.Truncate(1)
	require.NoError(t, err)

	_, err = log.Read(0) // the 0th record should be deleted
	require.Error(t, err)
	require.NoError(t, log.Close())
}
