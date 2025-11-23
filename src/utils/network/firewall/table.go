package firewall

import "github.com/google/nftables"

// TableConfig defines the configuration for a complete table
type TableConfig struct {
	Name   string
	Family nftables.TableFamily
	Chains []ChainConfig // Now using the unified ChainConfig
}

const (
	FilterTable      = "filter"
	NATTable         = "nat"
	InputChain       = "INPUT"
	ForwardChain     = "FORWARD"
	OutputChain      = "OUTPUT"
	PreroutingChain  = "PREROUTING"
	PostRoutingChain = "POSTROUTING"
)

// Predefined standard table configurations
var (
	StandardFilterTable = TableConfig{
		Name:   FilterTable,
		Family: nftables.TableFamilyINet,
		Chains: []ChainConfig{
			{
				Name:     InputChain,
				Table:    FilterTable,
				Create:   true,
				Type:     &[]nftables.ChainType{nftables.ChainTypeFilter}[0],
				Hook:     nftables.ChainHookInput,
				Priority: nftables.ChainPriorityFilter,
				Policy:   getChainPolicyAccept(),
			},
			{
				Name:     ForwardChain,
				Table:    FilterTable,
				Create:   true,
				Type:     &[]nftables.ChainType{nftables.ChainTypeFilter}[0],
				Hook:     nftables.ChainHookForward,
				Priority: nftables.ChainPriorityFilter,
				Policy:   getChainPolicyAccept(),
			},
			{
				Name:     OutputChain,
				Table:    FilterTable,
				Create:   true,
				Type:     &[]nftables.ChainType{nftables.ChainTypeFilter}[0],
				Hook:     nftables.ChainHookOutput,
				Priority: nftables.ChainPriorityFilter,
				Policy:   getChainPolicyAccept(),
			},
		},
	}

	StandardNATTable = TableConfig{
		Name:   NATTable,
		Family: nftables.TableFamilyIPv4,
		Chains: []ChainConfig{
			{
				Name:     PreroutingChain,
				Table:    NATTable,
				Create:   true,
				Type:     &[]nftables.ChainType{nftables.ChainTypeNAT}[0],
				Hook:     nftables.ChainHookPrerouting,
				Priority: nftables.ChainPriorityNATDest,
				Policy:   getChainPolicyAccept(),
			},
			{
				Name:     PostRoutingChain,
				Table:    NATTable,
				Create:   true,
				Type:     &[]nftables.ChainType{nftables.ChainTypeNAT}[0],
				Hook:     nftables.ChainHookPostrouting,
				Priority: nftables.ChainPriorityNATSource,
				Policy:   getChainPolicyAccept(),
			},
		},
	}
)

// CreateTableFromConfig creates a table and its chains based on the provided configuration
func CreateTableFromConfig(conn *nftables.Conn, config TableConfig) error {
	// Check if table already exists
	tables, tablesErr := conn.ListTables()
	if tablesErr != nil {
		return tablesErr
	}

	var table *nftables.Table
	for _, t := range tables {
		if t.Name == config.Name && t.Family == config.Family {
			table = t
			break
		}
	}

	// Create table if it doesn't exist
	if table == nil {
		table = &nftables.Table{
			Name:   config.Name,
			Family: config.Family,
		}
		conn.AddTable(table)
	}

	// Create chains
	if err := createChainsFromConfig(conn, table, config.Chains); err != nil {
		return err
	}

	// Flush all changes
	return conn.Flush()
}

// CreateStandardFilterTable creates the standard filter table with INPUT, FORWARD, OUTPUT chains
func CreateStandardFilterTable(conn *nftables.Conn) error {
	return CreateTableFromConfig(conn, StandardFilterTable)
}

// CreateStandardNATTable creates the standard NAT table with PREROUTING, POSTROUTING chains
func CreateStandardNATTable(conn *nftables.Conn) error {
	return CreateTableFromConfig(conn, StandardNATTable)
}

// EnsureStandardFirewallInfrastructure creates both filter and NAT tables
func EnsureStandardFirewallInfrastructure(conn *nftables.Conn) error {
	if err := CreateStandardFilterTable(conn); err != nil {
		return err
	}
	return CreateStandardNATTable(conn)
}

// createChainsFromConfig creates chains using the awesome NewChain function
func createChainsFromConfig(conn *nftables.Conn, table *nftables.Table, chainConfigs []ChainConfig) error {
	// Use the awesome NewChain function for each chain
	for _, chainConfig := range chainConfigs {
		// Convert ChainConfig to NewChain options
		opts := []Option{
			WithName(chainConfig.Name),
			WithinTable(table.Name),
		}

		// Add creation options if this is for chain creation
		if chainConfig.Create {
			opts = append(opts, Create())
			if chainConfig.Type != nil {
				opts = append(opts, WithChainType(*chainConfig.Type))
			}
			if chainConfig.Hook != nil {
				opts = append(opts, WithHook(chainConfig.Hook))
			}
			if chainConfig.Priority != nil {
				opts = append(opts, WithPriority(chainConfig.Priority))
			}
			if chainConfig.Policy != nil {
				opts = append(opts, WithPolicy(chainConfig.Policy))
			}
		}

		// Use the awesome NewChain function!
		_, _, err := NewChain(opts...)
		if err != nil {
			return err
		}
	}

	return nil
}

// getChainPolicyAccept returns a pointer to ChainPolicyAccept
func getChainPolicyAccept() *nftables.ChainPolicy {
	policy := nftables.ChainPolicyAccept
	return &policy
}
