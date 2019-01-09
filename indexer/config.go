package indexer

import (
	"plugin"

	"github.com/dexon-foundation/dexon/core"
	"github.com/dexon-foundation/dexon/dex/downloader"
)

// Config is data sources related configs struct.
type Config struct {
	// Used by dex/backend init flow.
	Enable bool

	// Plugin path for building components.
	Plugin string

	// PluginFlags for construction if needed.
	PluginFlags string

	// The genesis block from dex.Config
	Genesis *core.Genesis

	// Protocol options from dex.Config (partial)
	NetworkID uint64
	SyncMode  downloader.SyncMode
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
