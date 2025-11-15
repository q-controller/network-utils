package dhcp

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/coredhcp/coredhcp/config"
	"github.com/coredhcp/coredhcp/plugins"
	pl_dns "github.com/coredhcp/coredhcp/plugins/dns"
	pl_range "github.com/coredhcp/coredhcp/plugins/range"
	pl_router "github.com/coredhcp/coredhcp/plugins/router"
	pl_serverid "github.com/coredhcp/coredhcp/plugins/serverid"
	"github.com/coredhcp/coredhcp/server"
)

func StartDHCPServer(options ...DHCPOption) (*DHCPServer, error) {
	DHCPConfig := &DHCPConfig{
		LeaseTime: 12 * time.Hour,
		LeaseFile: "/tmp/qcontroller-dhcp-leases",
	}

	for _, opt := range options {
		if err := opt(DHCPConfig); err != nil {
			return nil, fmt.Errorf("failed to apply DHCP option: %v", err)
		}
	}

	if err := DHCPConfig.validate(); err != nil {
		return nil, fmt.Errorf("DHCP configuration validation failed: %v", err)
	}

	start := DHCPConfig.RangeStart
	end := DHCPConfig.RangeEnd

	var desiredPlugins = []*plugins.Plugin{
		&pl_dns.Plugin,
		&pl_range.Plugin,
		&pl_serverid.Plugin,
		&pl_router.Plugin,
	}

	for _, plugin := range desiredPlugins {
		if err := plugins.RegisterPlugin(plugin); err != nil {
			log.Fatalf("Failed to register plugin '%s': %v", plugin.Name, err)
		}
	}

	// Create configuration for DHCP server
	cfg := config.New()
	dnsArgs := make([]string, len(DHCPConfig.DNS))
	for i, ip := range DHCPConfig.DNS {
		dnsArgs[i] = ip.String()
	}

	cfg.Server4 = &config.ServerConfig{
		Addresses: []net.UDPAddr{
			{
				IP:   net.IPv4zero,
				Port: 67,
			},
		},
		Plugins: []config.PluginConfig{
			{Name: "server_id", Args: []string{DHCPConfig.Router.String()}},
			{Name: "range", Args: []string{DHCPConfig.LeaseFile, start.String(), end.String(), DHCPConfig.LeaseTime.String()}},
			{Name: "router", Args: []string{DHCPConfig.Router.String()}},
			{Name: "dns", Args: dnsArgs},
		},
	}

	srv, err := server.Start(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to start DHCP server: %v", err)
	}

	dhcpServer := &DHCPServer{
		server:   srv,
		done:     make(chan struct{}),
		stop:     make(chan struct{}),
		stopOnce: sync.Once{},
	}

	go func() {
		defer close(dhcpServer.done)
		<-dhcpServer.stop
		dhcpServer.server.Close()

		if waitErr := dhcpServer.server.Wait(); waitErr != nil {
			log.Printf("DHCP server error: %v", waitErr)
		}
	}()

	return dhcpServer, nil
}
