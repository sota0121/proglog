package server

import (
	"context"
	"net"
	"os"
	"testing"

	api "github.com/sota0121/proglog/api/v1"
	"github.com/sota0121/proglog/internal/log"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func TestServer(t *testing.T) {
	// Create a test scenario table.
	testMap := map[string]func(
		t *testing.T,
		client api.LogClient,
		config *Config,
	){
		"produce/consume a message to/from the log succeeds": testProduceConsume,
		"produce/consume stream succeeds":                    testProduceConsumeStream,
		"consume past log boundary fails":                    testConsumePastLogBoundary,
	}

	// Run each test scenario.
	for scenario, fn := range testMap {
		t.Run(scenario, func(t *testing.T) {
			client, config, teardown := setupTest(t, nil)
			defer teardown()
			fn(t, client, config)
		})
	}
}

// setupTest creates a test server and client.
func setupTest(t *testing.T, fn func(*Config)) (
	client api.LogClient,
	cfg *Config,
	teardown func(),
) {
	t.Helper() // mark this function as a helper function

	// ----------------------------------------------
	// Create a new listener.
	// ----------------------------------------------
	l, err := net.Listen("tcp", ":0") // 0 means to use any available port
	require.NoError(t, err)

	// ----------------------------------------------
	// Create a new client which connects to the server with insecure credentials.
	// ----------------------------------------------
	clientOptions := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	cc, err := grpc.Dial(l.Addr().String(), clientOptions...) // create a client connection
	require.NoError(t, err)

	// ----------------------------------------------
	// Create a new server.
	// ----------------------------------------------
	dir, err := os.MkdirTemp("", "server-test")
	require.NoError(t, err)

	clog, err := log.NewLog(dir, log.Config{})
	require.NoError(t, err)

	cfg = &Config{
		CommitLog: clog,
	}
	if fn != nil {
		fn(cfg)
	}
	server, err := NewGRPCServer(cfg)
	require.NoError(t, err)

	// ----------------------------------------------
	// Start the server as a goroutine
	// so that it doesn't block the test.
	// ----------------------------------------------
	go func() {
		server.Serve(l)
	}()

	// ----------------------------------------------
	// Create a new client.
	// ----------------------------------------------
	client = api.NewLogClient(cc)

	// Return the client, config, and a teardown function.
	return client, cfg, func() {
		cc.Close()    // close the client connection
		server.Stop() // stop the server
		l.Close()     // close the listener
		clog.Remove() // remove the log directory
	}

}

func testProduceConsume(t *testing.T, client api.LogClient, config *Config) {
	ctx := context.Background()

	// Arrange
	want := &api.Record{
		Value: []byte("hello world"),
	}

	// Act - produce a message
	produce, err := client.Produce(
		ctx,
		&api.ProduceRequest{
			Record: want,
		},
	)
	require.NoError(t, err)

	// Arrange - set the offset to the produced message
	want.Offset = produce.Offset

	// Act - consume the message
	consume, err := client.Consume(
		ctx,
		&api.ConsumeRequest{
			Offset: produce.Offset,
		},
	)
	// Assert
	require.NoError(t, err)
	require.Equal(t, want.Value, consume.Record.Value)
	require.Equal(t, want.Offset, consume.Record.Offset)
}

func testProduceConsumeStream(t *testing.T, client api.LogClient, config *Config) {
	ctx := context.Background()

	// Arrange
	wants := []*api.Record{{
		Value:  []byte("first message"),
		Offset: 0,
	}, {
		Value:  []byte("second message"),
		Offset: 1,
	}}

	// Act - produce messages in a stream
	{
		// Get a stream to produce messages.
		stream, err := client.ProduceStream(ctx)
		require.NoError(t, err)

		// Send messages to the stream.
		for offset, want := range wants {
			// Send a message to the stream.
			err = stream.Send(&api.ProduceRequest{
				Record: want,
			})
			require.NoError(t, err)

			// Receive a response from the stream.
			res, err := stream.Recv()
			require.NoError(t, err)

			// Assert - the sending data must be the same as the received data.
			require.Equal(t, uint64(offset), res.Offset)
			if res.Offset != uint64(offset) {
				t.Fatalf(
					"got offset: %d, want: %d",
					res.Offset,
					offset,
				)
			}
		}

	}

	// Act - consume messages in a stream
	{
		// Get a stream to consume messages.
		stream, err := client.ConsumeStream(
			ctx,
			&api.ConsumeRequest{Offset: 0},
		)
		require.NoError(t, err)

		for offset, want := range wants {
			// Receive a message from the stream.
			res, err := stream.Recv()
			require.NoError(t, err)

			// Assert - the received data must be the same as the sending data.
			require.Equal(t, res.Record, &api.Record{
				Value:  want.Value,
				Offset: uint64(offset),
			})
		}
	}
}

func testConsumePastLogBoundary(t *testing.T, client api.LogClient, config *Config) {
	ctx := context.Background()

	// Arrange - produce a message
	produce, err := client.Produce(ctx, &api.ProduceRequest{
		Record: &api.Record{
			Value: []byte("hello world"),
		},
	})
	require.NoError(t, err)

	// Act - consume the message which is out of range
	consume, err := client.Consume(ctx, &api.ConsumeRequest{
		Offset: produce.Offset + 1,
	})
	if consume != nil {
		t.Fatal("expected consume to be nil")
	}
	got := status.Code(err)
	want := status.Code(api.ErrOffsetOutOfRange{}.GRPCStatus().Err())
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}
