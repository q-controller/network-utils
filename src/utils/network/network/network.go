package network

type Network interface {
	Destroy() error

	Execute(func() error) error

	Connect(iface string, masquerade bool) error
	Disconnect(iface string, masquerade bool) error
}
