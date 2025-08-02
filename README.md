# Network Utilities
A set of tools for easily creating network bridges and TAP interfaces, with a focus on configuring bridge networking to enable seamless communication between devices (TAP), the host, and the internet. Internally, the tool uses `nftables` for firewall and NAT configuration, as `iptables` is deprecated and superseded by `nftables` in modern Linux systems due to its improved performance, scalability, and better integration with modern Linux kernels.

When configuring a bridge for communication, the following steps are performed to ensure traffic is not blocked by firewalls such as UFW:

1. **Masquerading on the Host Interface:**
    Network Address Translation (NAT) masquerading is applied to the host's outbound interface. This allows devices connected to the bridge to access the internet using the host's IP address, ensuring return traffic is properly routed back to the originating device.

2. **Forwarding Rules for Outbound and Return Traffic:**
    Dedicated forwarding rules are created to allow both outbound traffic from bridge-connected devices to the internet and return traffic from the internet back to those devices. This is essential for full bidirectional connectivity.

3. **Allowing DHCP/DNS Traffic:**
    Rules are added to permit DHCP and DNS traffic, enabling devices on the bridge to obtain IP addresses and resolve domain names without restriction.

To avoid conflicts and restrictions imposed by UFW (Uncomplicated Firewall, a popular Linux firewall management tool), these rules are not placed directly in the default `forward` or `input` chains. Instead, new dedicated chains are created, and traffic is explicitly jumped to these chains. These custom chains are inserted before the chains managed by UFW, ensuring that bridge-related traffic is handled correctly and not inadvertently blocked.

This approach allows the software to provide robust and reliable bridge networking, bypassing common firewall limitations and enabling transparent communication between devices, the host, and the internet.

## Features

- Create and manage network bridges
- Set up TAP interfaces
- Configure bridge networking for device-to-device, host, and internet communication

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

## Tests

To run the tests, use the following command:

```shell
go test -v ./... | go tool github.com/jstemmer/go-junit-report
```

This command executes all tests in the project and generates JUnit XML output.
