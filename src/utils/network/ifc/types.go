package ifc

import "net"

type LinkType string

const (
	LinkTypeBridge LinkType = "bridge"
	LinkTypeTap    LinkType = "tap"
)

type LinkManager interface {
	AddLink(name string, linkType LinkType) error
	SetIP(name string, ip net.IP, mask net.IPMask) error
	Exists(name string) (bool, error)
	SetMaster(name string, masterName string) error
	BringUp(name string) error
	HasIP(name string, ip net.IP, mask net.IPMask) (bool, error)
	DeleteLink(name string) error
	DisableTxOffloading(name string) error
}
