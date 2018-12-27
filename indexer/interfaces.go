package indexer

// NewIndexerFuncName plugin looks up name.
var NewIndexerFuncName = "NewIndexer"

// NewIndexerFunc init function alias.
type NewIndexerFunc = func(ReadOnlyBlockChain, Config) Indexer

// Indexer defines indexer daemon interface. The daemon would hold a
// core.Blockhain, passed by initialization function, to receiving latest block
// event or other information query and interaction.
type Indexer interface {
	// Start is called by dex.Dexon if config is set.
	Start() error

	// Stop is called by dex.Dexon if config is set and procedure is
	// terminating.
	Stop() error
}
