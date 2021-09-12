package xds

import (
	"context"
	"flag"
	"os"

	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"github.com/envoyproxy/go-control-plane/pkg/test/v3"
	"github.com/nfowl/quilkin-controller/internal/store"
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
	nodeVersions = make(map[string]int)
}

type CacheUpdater struct {
	cache   cachev3.SnapshotCache
	updates chan store.NodeConfig
	deletes chan string
	logger  *zap.SugaredLogger
}

func (c *CacheUpdater) handleUpdates() {
	c.logger.Info("Starting Cache Update handler")
	for update := range c.updates {
		c.logger.Info("Update snapshot for node: ", update.ProxyName)
		snap := generateNodeSnapshot(update)
		if err := snap.Consistent(); err != nil {
			c.logger.Errorf("snapshot inconsistency: %s", err)
			os.Exit(1)
		}
		// c.logger.Infof("will serve snapshot %+v", snap)
		if err := c.cache.SetSnapshot(update.ProxyName, snap); err != nil {
			c.logger.Errorf("snapshot error %q for %+v", err, snap)
			os.Exit(1)
		}
	}
}

func (c *CacheUpdater) handleDeletes() {
	c.logger.Info("Starting Cache Deletion handler")
	for delete := range c.deletes {
		c.logger.Info("Deleting snapshots for node: %s", delete)
		c.cache.ClearSnapshot(delete)
	}
}

func StartServer(l *zap.SugaredLogger, updates chan store.NodeConfig, deletes chan string) {
	cache := cachev3.NewSnapshotCache(false, cachev3.IDHash{}, l)
	updater := CacheUpdater{cache: cache, updates: updates, deletes: deletes, logger: l}
	// Run the xDS server
	ctx := context.Background()
	cb := &test.Callbacks{Debug: false}
	srv := server.NewServer(ctx, cache, cb)
	go RunServer(ctx, srv, port)
	go updater.handleDeletes()
	go updater.handleUpdates()
}
