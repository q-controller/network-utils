package network

type Network interface {
	Destroy() error

	Execute(func() error) error

	Connect(hostIf string) error
	Disconnect(hostIf string) error
}
