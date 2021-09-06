package xds

import (
	"context"
	"flag"
	"os"

	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"github.com/envoyproxy/go-control-plane/pkg/test/v3"
	"go.uber.org/zap"
)

var (
	port   uint
	nodeID string
)

func init() {
	// The port that this xDS server listens on
	flag.UintVar(&port, "port", 18000, "xDS management server port")

	// Tell Envoy to use this Node ID
	flag.StringVar(&nodeID, "nodeID", "proxy-1", "Node ID")
}

func StartServer(l *zap.SugaredLogger) {
	cache := cachev3.NewSnapshotCache(false, cachev3.IDHash{}, l)

	// Create the snapshot that we'll serve to Envoy
	snapshot := GenerateSnapshot()
	if err := snapshot.Consistent(); err != nil {
		l.Errorf("snapshot inconsistency: %s", err)
		os.Exit(1)
	}
	l.Debugf("will serve snapshot %+v", snapshot)

	// Add the snapshot to the cache
	if err := cache.SetSnapshot(nodeID, snapshot); err != nil {
		l.Errorf("snapshot error %q for %+v", err, snapshot)
		os.Exit(1)
	}

	// Run the xDS server
	ctx := context.Background()
	cb := &test.Callbacks{Debug: false}
	srv := server.NewServer(ctx, cache, cb)
	go RunServer(ctx, srv, port)
}
