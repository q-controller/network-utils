package dns

import (
	"context"
	"log/slog"
	"net"
	"time"
)

var _ DNSForwarder = (*ManagedForwarder)(nil)

// InterfaceProber checks whether the target network interface is present.
type InterfaceProber interface {
	Probe() (net.IP, error)
}

// ForwarderFactory creates a DNS forwarder bound to the given address.
type ForwarderFactory interface {
	NewForwarder(ctx context.Context, addr string) (DNSForwarder, error)
}

// ManagedForwarder tracks an interface lifecycle and starts/stops a DNS
// forwarder as the interface appears and disappears.
type ManagedForwarder struct {
	ctx      context.Context
	cancel   context.CancelFunc
	done     chan struct{}
	interval time.Duration
	prober   InterfaceProber
	factory  ForwarderFactory
}

func NewManagedForwarder(
	parent context.Context,
	interval time.Duration,
	prober InterfaceProber,
	factory ForwarderFactory,
) *ManagedForwarder {
	ctx, cancel := context.WithCancel(parent)
	return &ManagedForwarder{
		ctx:      ctx,
		cancel:   cancel,
		done:     make(chan struct{}),
		interval: interval,
		prober:   prober,
		factory:  factory,
	}
}

func (m *ManagedForwarder) Serve() (func(), error) {
	go m.run()
	stop := func() {
		m.cancel()
		<-m.done
	}
	return stop, nil
}

func (m *ManagedForwarder) run() {
	defer close(m.done)
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	var stopForwarder func()

	for {
		select {
		case <-m.ctx.Done():
			if stopForwarder != nil {
				stopForwarder()
			}
			return
		case <-ticker.C:
			if stopForwarder == nil {
				hostIP, err := m.prober.Probe()
				if err != nil {
					slog.Debug("Waiting for interface", "error", err)
					continue
				}
				f, fErr := m.factory.NewForwarder(m.ctx, hostIP.String())
				if fErr != nil {
					slog.Error("Failed to start DNS forwarder", "address", hostIP, "error", fErr)
					continue
				}
				stop, sErr := f.Serve()
				if sErr != nil {
					slog.Error("Failed to serve DNS forwarder", "address", hostIP, "error", sErr)
					continue
				}
				slog.Info("DNS forwarder started", "address", hostIP)
				stopForwarder = stop
			} else {
				if _, err := m.prober.Probe(); err != nil {
					slog.Info("Interface disappeared, stopping DNS forwarder")
					stopForwarder()
					stopForwarder = nil
				}
			}
		}
	}
}
