![Build and Test Go Project](https://github.com/q-controller/network-utils/actions/workflows/test.yml/badge.svg)

# Network Utilities
A set of tools for easily creating network bridges and TAP interfaces, with a focus on configuring bridge networking to enable seamless communication between devices (TAP), the host, and the internet. Internally, the tool uses `nftables` for firewall and NAT configuration, as `iptables` is deprecated and superseded by `nftables` in modern Linux systems due to its improved performance, scalability, and better integration with modern Linux kernels.

When configuring a bridge for communication, the following steps are performed to ensure traffic is not blocked by firewalls such as UFW:

1. **Ensure standard nftables tables exist:**
    The `inet filter` and `ip nat` tables (with the standard `FORWARD`, `INPUT`, `OUTPUT`, `PREROUTING`, `POSTROUTING` chains) are created if missing. This makes the library work on pristine hosts (e.g. fresh cloud-init VMs) where no nftables ruleset has been populated yet. The operation is idempotent — existing tables on hosts where Docker, firewalld or iptables-nft has already created them are left untouched.

2. **Masquerading on the Host Interface:**
    Network Address Translation (NAT) masquerading is applied to the host's outbound interface. This allows devices connected to the bridge to access the internet using the host's IP address, ensuring return traffic is properly routed back to the originating device.

3. **Reverse masquerading on the bridge-side interface:**
    Connections initiated from outside the bridge subnet (host → VM, or external → VM via a route through this host) are SNAT'd to the bridge-side veth IP. The VM sees its directly-attached gateway as the source and can always reply; conntrack restores the original source on the way back out. Without this, the VM would try to reply directly to an external IP it has no route for, and the reply would be dropped somewhere along the path.

4. **Forwarding Rules for Outbound and Return Traffic:**
    Dedicated forwarding rules are created to allow outbound traffic from bridge-connected devices to the internet and return traffic from the internet back to those devices. The return-traffic rule requires `ct state {established,related}`, so only replies to flows the bridge originated are accepted — unsolicited external→VM packets are not.

5. **Allowing DHCP/DNS Traffic:**
    Rules are added to permit DHCP and DNS traffic, enabling devices on the bridge to obtain IP addresses and resolve domain names without restriction.

To avoid conflicts and restrictions imposed by UFW (Uncomplicated Firewall, a popular Linux firewall management tool), these rules are not placed directly in the default `forward` or `input` chains. Instead, new dedicated chains are created, and traffic is explicitly jumped to these chains. These custom chains are inserted before the chains managed by UFW, ensuring that bridge-related traffic is handled correctly and not inadvertently blocked.

This approach allows the software to provide robust and reliable bridge networking, bypassing common firewall limitations and enabling transparent communication between devices, the host, and the internet.

### Operator prerequisites

This library configures firewall rules but deliberately does **not** touch system-wide kernel knobs. The operator is responsible for setting:

- **`net.ipv4.ip_forward=1`** — required for the host to route packets between the bridge and the uplink (i.e. for VMs to reach the internet, or for external hosts to reach VMs). Persist it via `/etc/sysctl.d/`:

  ```
  # /etc/sysctl.d/99-qcontroller.conf
  net.ipv4.ip_forward = 1
  ```

  Apply without reboot: `sudo sysctl --system`.

  This is left to the operator because `ip_forward` is a host-wide setting that affects every workload on the machine, and hardened hosts may have it disabled on purpose.

## Features

- Create and manage network bridges
- Set up TAP interfaces
- Configure bridge networking for device-to-device, host, and internet communication
- DHCP server for automatic IP assignment to bridge-connected devices
- DNS forwarding with failover support and optional CoreDNS backend

## Build

Ensure that [Go](https://golang.org/dl/) is installed and properly configured on your system before proceeding.

To build the project, run:

```sh
go build
```

This will produce the `network-utils` executable in the current directory.

## Usage

Basic usage examples:

```shell
# Create a bridge with a specific name and subnet
# The `--disable-tx-offload` flag disables TX Checksum Offload.
./network-utils create-bridge --name br0 --cidr 192.168.26.0/24 --disable-tx-offload

# Attach the bridge to a host network interface (e.g., `eth0` for Ethernet or `wlan0` for Wi-Fi)
./network-utils configure-bridge --name br0 --hostIf wlan0

# Create a TAP interface and add it to the bridge
./network-utils create-tap --name tap0 --bridge br0
```

To use the TAP device with a QEMU VM:

```sh
qemu-system-x86_64 -machine q35 -accel kvm -m 960 -nographic \
    -device virtio-net,netdev=net0,mac=2e:c8:40:59:7d:16 \
    -netdev tap,id=net0,ifname=tap0,script=no,downscript=no \
    -qmp unix:/tmp/test.sock,server,wait=off -cpu host -smp 1 -hda <IMAGE>
```

After starting the VM, it will have internet access and be reachable from the host.

## DHCP

A tiny abstraction over [CoreDHCP](https://github.com/coredhcp/coredhcp) for running a DHCP server on the bridge interface. See [dhcp/README.md](src/utils/network/dhcp/README.md) for details.

## DNS Forwarding

The DNS package provides two forwarding implementations that listen on a bridge/gateway IP and forward queries to upstream resolvers:

- **`DNSFailoverForwarder`** — a lightweight custom forwarder that tries upstream servers in order and returns the first successful response. Supports dynamic upstream discovery from `resolv.conf` (with file watching) or static upstreams.
- **`CoreDNSServer`** — uses embedded [CoreDNS](https://coredns.io/) with the `forward` plugin, useful on systems without systemd-resolved where CoreDNS can handle `resolv.conf` reloading and plugin-based extensibility natively.

### Static upstreams (proxy mode)

On systems with systemd-resolved, the forwarder can be configured with a static upstream pointing to `127.0.0.53`. This delegates all resolution — including `/etc/hosts`, mDNS, split-DNS, and VPN configurations — to systemd-resolved:

```go
forwarder, err := dns.NewDNSFailoverForwarder(ctx,
    dns.WithForwarderAddress(gatewayIP),
    dns.WithForwarderTimeout(2*time.Second),
    dns.WithUpstreams([]string{"127.0.0.53:53"}),
)
```

### Dynamic upstreams (resolv.conf watching)

When no static upstreams are provided, the forwarder watches a `resolv.conf` file for changes and updates its upstream list automatically. It prefers `/run/systemd/resolve/resolv.conf` when available, falling back to `/etc/resolv.conf`:

```go
forwarder, err := dns.NewDNSFailoverForwarder(ctx,
    dns.WithForwarderAddress(gatewayIP),
    dns.WithForwarderTimeout(2*time.Second),
)
```

The path can be overridden with `dns.WithResolvconfPath(path)`.

### CoreDNS backend

For non-systemd systems or when CoreDNS plugins are needed:

```go
server, err := dns.NewCoreDNSServer(ctx,
    dns.WithForwarderAddress(gatewayIP),
    dns.WithResolvconfPath("/etc/resolv.conf"),
)
```

## Tests

To run the tests, use the following command:

```shell
go test -v ./... | go tool github.com/jstemmer/go-junit-report
```

This command executes all tests in the project and generates JUnit XML output.
