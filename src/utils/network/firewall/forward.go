//go:build linux
// +build linux

package firewall

import (
	"github.com/google/nftables"
	"github.com/google/nftables/binaryutil"
	"github.com/google/nftables/expr"
)

func ForwardOutboundRule(chainName, tableName, hostIf, internalIf string) NewRule {
	return func(rules *Rules) error {
		chain, table, chainErr := NewChain(
			WithName(chainName),
			WithinTable(tableName),
		)
		if chainErr != nil {
			return chainErr
		}

		rule := &nftables.Rule{
			Table: table,
			Chain: chain,
			Exprs: []expr.Any{
				// [ meta load iifname => reg 1 ]
				&expr.Meta{Key: expr.MetaKeyIIFNAME, Register: 1},
				// [ cmp eq reg 1 interfaceA ]
				&expr.Cmp{Op: expr.CmpOpEq, Register: 1, Data: []byte(internalIf + "\x00")},
				// [ meta load oifname => reg 2 ]
				&expr.Meta{Key: expr.MetaKeyOIFNAME, Register: 2},
				// [ cmp eq reg 2 interfaceB ]
				&expr.Cmp{Op: expr.CmpOpEq, Register: 2, Data: []byte(hostIf + "\x00")},
				// [ immediate verdict ACCEPT ]
				&expr.Verdict{Kind: expr.VerdictAccept},
			},
		}

		rules.rules = append(rules.rules, rule)
		return nil
	}
}

func ForwardReturnTrafficRule(chainName, tableName, hostIf, internalIf string) NewRule {
	return func(rules *Rules) error {
		chain, table, chainErr := NewChain(
			WithName(chainName),
			WithinTable(tableName),
		)
		if chainErr != nil {
			return chainErr
		}

		// ct state mask covering ESTABLISHED|RELATED. The kernel stores
		// ct state as a 4-byte bitmask; we mask it with the bits we accept
		// and require the result to be non-zero.
		ctStateMask := binaryutil.NativeEndian.PutUint32(
			expr.CtStateBitESTABLISHED | expr.CtStateBitRELATED,
		)

		rule := &nftables.Rule{
			Table: table,
			Chain: chain,
			Exprs: []expr.Any{
				// [ meta load iifname => reg 1 ]
				&expr.Meta{Key: expr.MetaKeyIIFNAME, Register: 1},
				// [ cmp eq reg 1 interfaceB ]
				&expr.Cmp{Op: expr.CmpOpEq, Register: 1, Data: []byte(hostIf + "\x00")},
				// [ meta load oifname => reg 2 ]
				&expr.Meta{Key: expr.MetaKeyOIFNAME, Register: 2},
				// [ cmp eq reg 2 interfaceA ]
				&expr.Cmp{Op: expr.CmpOpEq, Register: 2, Data: []byte(internalIf + "\x00")},
				// [ ct load state => reg 1 ]
				&expr.Ct{Register: 1, Key: expr.CtKeySTATE},
				// [ bitwise reg 1 = reg 1 & mask ^ 0 ]
				&expr.Bitwise{
					SourceRegister: 1,
					DestRegister:   1,
					Len:            4,
					Mask:           ctStateMask,
					Xor:            []byte{0, 0, 0, 0},
				},
				// [ cmp neq reg 1 0 ]
				&expr.Cmp{Op: expr.CmpOpNeq, Register: 1, Data: []byte{0, 0, 0, 0}},
				// [ immediate verdict ACCEPT ]
				&expr.Verdict{Kind: expr.VerdictAccept},
			},
		}

		rules.rules = append(rules.rules, rule)
		return nil
	}
}
