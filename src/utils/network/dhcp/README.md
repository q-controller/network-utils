# DHCP Package

This package is a tiny abstraction over [CoreDHCP](https://github.com/coredhcp/coredhcp), providing a simplified interface for setting up and managing DHCP servers in Go projects.

## Example Usage

Below is an example of how to use this package to start a DHCP server:

```go
dhcpServer, dhcpServerErr := dhcp.StartDHCPServer(
    dhcp.WithInterface(Name, net.ParseIP("192.168.26.1")),
    dhcp.WithDNS(net.ParseIP("8.8.8.8")),
    dhcp.WithLeaseTime(time.Second*time.Duration(12)),
    dhcp.WithRange(
        net.ParseIP("192.168.26.10"),
        net.ParseIP("192.168.26.20"),
    ),
)
```

This example demonstrates how to configure and start a DHCP server using the provided abstraction.
