package server

import (
	"net"
	"os"
	"testing"

	api "github.com/sota0121/proglog/api/v1"
	"github.com/sota0121/proglog/internal/log"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestServer(t *testing.T) {
	// Create a test scenario table.
	testMap := map[string]func(
		t *testing.T,
		client  api.LogClient,
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

	cfg := &Config{
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
		cc.Close() // close the client connection
		server.Stop() // stop the server
		l.Close() // close the listener
		clog.Remove() // remove the log directory
	}

}

func testProduceConsume(t *testing.T, client, api.LogClient, config *Config) {
}

func testProduceConsumeStream(t *testing.T, client, api.LogClient, config *Config) {
}

func testConsumePastLogBoundary(t *testing.T, client, api.LogClient, config *Config) {
}