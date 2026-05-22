//go:build linux
// +build linux

package firewall

import (
	"errors"
	"net"

	"github.com/google/nftables"
	"github.com/google/nftables/expr"
)

func MasqueradeRule(chainName, tableName, interfaceName string) NewRule {
	return func(rules *Rules) error {
		chain, table, chainErr := NewChain(
			WithName(chainName),
			WithinTable(tableName),
		)
		if chainErr != nil {
			return chainErr
		}

		masqRule := &nftables.Rule{
			Table: table,
			Chain: chain,
			Exprs: []expr.Any{
				// [ meta load oifname => reg 1 ]
				&expr.Meta{Key: expr.MetaKeyOIFNAME, Register: 1},
				// [ cmp eq reg 1 lanInterface ]
				&expr.Cmp{Op: expr.CmpOpEq, Register: 1, Data: []byte(interfaceName + "\x00")},
				// [ immediate verdict MASQUERADE ]
				&expr.Masq{},
			},
		}

		rules.rules = append(rules.rules, masqRule)
		return nil
	}
}

// ReverseMasqueradeRule SNATs incoming connections to the VM so that the
// VM sees this host (hostLink's IP) as the source instead of the real
// external caller. The VM can always reply to its directly attached
// gateway; it usually can't reply to an arbitrary external IP because it
// has no route for it. The original source is restored by conntrack on
// the way back out.
//
// vmSubnet excludes VM-originated traffic so it falls through to the
// outbound MasqueradeRule on the uplink instead of being SNAT'd here.
func ReverseMasqueradeRule(chainName, tableName, hostLink string, vmSubnet *net.IPNet) NewRule {
	return func(rules *Rules) error {
		if vmSubnet == nil {
			return errors.New("vm subnet is required for reverse masquerade rule")
		}
		ipv4 := vmSubnet.IP.To4()
		if ipv4 == nil || len(vmSubnet.Mask) != net.IPv4len {
			return errors.New("vm subnet must be IPv4")
		}

		chain, table, chainErr := NewChain(
			WithName(chainName),
			WithinTable(tableName),
		)
		if chainErr != nil {
			return chainErr
		}

		network := make([]byte, net.IPv4len)
		for i := 0; i < net.IPv4len; i++ {
			network[i] = ipv4[i] & vmSubnet.Mask[i]
		}

		masqRule := &nftables.Rule{
			Table: table,
			Chain: chain,
			Exprs: []expr.Any{
				// [ meta load oifname => reg 1 ]
				&expr.Meta{Key: expr.MetaKeyOIFNAME, Register: 1},
				// [ cmp eq reg 1 hostLink ]
				&expr.Cmp{Op: expr.CmpOpEq, Register: 1, Data: []byte(hostLink + "\x00")},
				// [ payload load 4b @ network header + 12 => reg 1 ] (src IP)
				&expr.Payload{
					DestRegister: 1,
					Base:         expr.PayloadBaseNetworkHeader,
					Offset:       12,
					Len:          4,
				},
				// [ bitwise reg 1 = reg 1 & subnet.Mask ^ 0 ]
				&expr.Bitwise{
					SourceRegister: 1,
					DestRegister:   1,
					Len:            4,
					Mask:           []byte(vmSubnet.Mask),
					Xor:            []byte{0, 0, 0, 0},
				},
				// [ cmp neq reg 1 network_addr ]
				&expr.Cmp{Op: expr.CmpOpNeq, Register: 1, Data: network},
				// [ immediate verdict MASQUERADE ]
				&expr.Masq{},
			},
		}

		rules.rules = append(rules.rules, masqRule)
		return nil
	}
}
