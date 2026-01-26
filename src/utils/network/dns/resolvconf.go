package dns

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/miekg/dns"
)

const (
	ResolvConfPath        = "/etc/resolv.conf"
	SystemdResolvConfPath = "/run/systemd/resolve/resolv.conf"
)

type UpstreamDNS struct {
	Endpoints []string
	Error     error
}

func readUpstreams(filename string) UpstreamDNS {
	config, configErr := dns.ClientConfigFromFile(filename)
	upstreams := UpstreamDNS{
		Error:     configErr,
		Endpoints: []string{},
	}
	if configErr == nil {
		for _, server := range config.Servers {
			upstreams.Endpoints = append(upstreams.Endpoints, net.JoinHostPort(server, config.Port))
		}
	}
	return upstreams
}

func GetUpstreamDNSFromFile(ctx context.Context, filename string) (<-chan UpstreamDNS, error) {
	ch := make(chan UpstreamDNS)

	watcher, watcherErr := fsnotify.NewWatcher()
	if watcherErr != nil {
		return nil, watcherErr
	}

	dir := filepath.Dir(filename)
	base := filepath.Base(filename)

	if addErr := watcher.Add(dir); addErr != nil {
		return nil, addErr
	}

	slog.Debug("WatchFileChanges: starting to watch dir", "dir", dir)
	go func() {
		defer func() {
			if closeErr := watcher.Close(); closeErr != nil {
				slog.Error("WatchFileChanges: failed to close watcher", "error", closeErr)
			}
		}()

		prev := []string{}
		ups := readUpstreams(filename)
		prev = ups.Endpoints
		ch <- ups
		defer close(ch)
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					ch <- UpstreamDNS{
						Endpoints: nil,
						Error:     fmt.Errorf("watcher closed"),
					}
					return
				}
				slog.Debug("WatchFileChanges: received event", "event", event.Op, "name", event.Name)
				if filepath.Base(event.Name) == base {
					curr := readUpstreams(filename)
					if !Same(prev, curr.Endpoints) {
						ch <- curr
						prev = curr.Endpoints
					}
				}
			case _, ok := <-watcher.Errors:
				if !ok {
					ch <- UpstreamDNS{
						Endpoints: nil,
						Error:     fmt.Errorf("watcher error channel closed"),
					}
					return
				}
			case <-ctx.Done():
				ch <- UpstreamDNS{
					Endpoints: nil,
					Error:     ctx.Err(),
				}
				return
			}
		}
	}()

	return ch, nil
}
