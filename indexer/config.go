package indexer

import (
	"plugin"
)

// Config is data sources related configs struct.
type Config struct {
	// Used by dex/backend init flow.
	Enable bool

	// Plugin path for building components.
	Plugin string

	// PluginFlags for construction if needed.
	PluginFlags string
}

// NewIndexerFromConfig initialize exporter according to given config.
func NewIndexerFromConfig(bc ReadOnlyBlockChain, c Config) (idx Indexer) {
	if c.Plugin == "" {
		// default
		return
	}

	plug, err := plugin.Open(c.Plugin)
	if err != nil {
		panic(err)
	}

	symbol, err := plug.Lookup(NewIndexerFuncName)
	if err != nil {
		panic(err)
	}

	idx = symbol.(NewIndexerFunc)(bc, c)
	return
}
