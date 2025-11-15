package dhcp

import (
	"sync"

	"github.com/coredhcp/coredhcp/server"
)

type DHCPServer struct {
	server   *server.Servers
	done     chan struct{}
	stop     chan struct{}
	stopOnce sync.Once
}

func (ds *DHCPServer) Stop() {
	ds.stopOnce.Do(func() {
		close(ds.stop)
	})
}

func (d *DHCPServer) Done() <-chan struct{} {
	return d.done
}
