//go:build linux
// +build linux

package firewall

import (
	"fmt"

	"github.com/google/nftables"
)

type ChainConfig struct {
	Name   string
	Table  string
	Create bool
	// Rich configuration options (optional - for creating new chains)
	Type     *nftables.ChainType
	Hook     *nftables.ChainHook
	Priority *nftables.ChainPriority
	Policy   *nftables.ChainPolicy
}

type Option func(*ChainConfig)

func NewChain(opts ...Option) (*nftables.Chain, *nftables.Table, error) {
	conn := getConnection()

	config := &ChainConfig{}
	for _, opt := range opts {
		opt(config)
	}

	if config.Name == "" || config.Table == "" {
		return nil, nil, fmt.Errorf("chain name and table must be specified")
	}

	tables, tablesErr := conn.ListTables()
	if tablesErr != nil {
		return nil, nil, tablesErr
	}

	for _, table := range tables {
		if table.Name == config.Table {
			// Table exists, use it
			chains, chainsErr := conn.ListChains()
			if chainsErr != nil {
				return nil, nil, chainsErr
			}

			for _, ch := range chains {
				if ch.Name == config.Name {
					return ch, table, nil // Chain already exists, return it
				}
			}

			if config.Create {
				// Chain does not exist, create it
				customChain := &nftables.Chain{
					Name:  config.Name,
					Table: table,
				}

				// Apply rich configuration if provided
				if config.Type != nil {
					customChain.Type = *config.Type
				}
				if config.Hook != nil {
					customChain.Hooknum = config.Hook
				}
				if config.Priority != nil {
					customChain.Priority = config.Priority
				}
				if config.Policy != nil {
					customChain.Policy = config.Policy
				}

				customChain = conn.AddChain(customChain)
				if err := conn.Flush(); err != nil {
					return nil, nil, err
				}
				return customChain, table, nil // New chain created
			}

			return nil, nil, fmt.Errorf("chain %s does not exist in table %s", config.Name, config.Table)
		}
	}

	return nil, nil, fmt.Errorf("table %s does not exist", config.Table)
}

func WithName(chainName string) Option {
	return func(config *ChainConfig) {
		config.Name = chainName // Set the name for the chain
	}
}

func WithinTable(tableName string) Option {
	return func(config *ChainConfig) {
		config.Table = tableName // Set the table name for the chain
	}
}

func Create() Option {
	return func(config *ChainConfig) {
		config.Create = true // This option indicates that the chain should be created if it does not already exists
	}
}

// Rich configuration options for creating new chains

func WithChainType(chainType nftables.ChainType) Option {
	return func(config *ChainConfig) {
		config.Type = &chainType
	}
}

func WithHook(hook *nftables.ChainHook) Option {
	return func(config *ChainConfig) {
		config.Hook = hook
	}
}

func WithPriority(priority *nftables.ChainPriority) Option {
	return func(config *ChainConfig) {
		config.Priority = priority
	}
}

func WithPolicy(policy *nftables.ChainPolicy) Option {
	return func(config *ChainConfig) {
		config.Policy = policy
	}
}
