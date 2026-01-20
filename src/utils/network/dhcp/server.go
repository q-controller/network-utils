package dhcp

import (
	"log/slog"
	"os"
	"sync"

	"github.com/coredhcp/coredhcp/server"
)

type DHCPServer struct {
	server    *server.Servers
	stopOnce  sync.Once
	leaseFile string
}

func (ds *DHCPServer) Stop() {
	ds.stopOnce.Do(func() {
		defer func() {
			if err := os.Remove(ds.leaseFile); err != nil {
				slog.Info("Failed to remove lease file", "error", err)
			}
		}()
		slog.Info("Stopping DHCP server")
		ds.server.Close()

		if waitErr := ds.server.Wait(); waitErr != nil {
			slog.Info("DHCP server error", "error", waitErr)
		}
	})
}
