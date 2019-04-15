// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package params

import (
	"fmt"
	"math/big"

	"github.com/dexon-foundation/dexon/common"
	"github.com/dexon-foundation/dexon/common/math"
)

// Genesis hashes to enforce below configs on.
var (
	MainnetGenesisHash = common.HexToHash("0xc8f6f69570e8eb740f3d74995702f4fc4dfa613ec56a0cebf941cc035606099b")
	TestnetGenesisHash = common.HexToHash("0x7d8700a7a731162880adff4f21398a901c0b75d907bec8f4eac51460f94cb846")
	TaipeiGenesisHash  = common.HexToHash("0x5929cb70fe4ba22dce821b2efca737a1874a0f5a34f3ffb9a1e157516622e20b")
	YilanGenesisHash   = common.HexToHash("0xdcdafc044c24d728c6149ecfada746d8de6e59fc5d18063caf7950badc1df12e")
)

// TrustedCheckpoints associates each known checkpoint with the genesis hash of
// the chain it belongs to.
var TrustedCheckpoints = map[common.Hash]*TrustedCheckpoint{
	MainnetGenesisHash: MainnetTrustedCheckpoint,
	TestnetGenesisHash: TestnetTrustedCheckpoint,
}

var (
	// MainnetChainConfig is the chain parameters to run a node on the main network.
	MainnetChainConfig = &ChainConfig{
		ChainID:             big.NewInt(237),
		DMoment:             1554142300,
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        big.NewInt(0),
		DAOForkSupport:      true,
		EIP150Block:         big.NewInt(0),
		EIP150Hash:          common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		Dexcon: &DexconConfig{
			GenesisCRSText:    "In DEXON, we trust.",
			Owner:             common.HexToAddress("BF8C48A620bacc46907f9B89732D25E47A2D7Cf7"),
			MinStake:          new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1e6)),
			LockupPeriod:      86400 * 1000,
			MiningVelocity:    0.1875,
			NextHalvingSupply: new(big.Int).Mul(big.NewInt(1e18), big.NewInt(2.5e9)),
			LastHalvedAmount:  new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1.5e9)),
			MinGasPrice:       new(big.Int).Mul(big.NewInt(1e9), big.NewInt(24)),
			BlockGasLimit:     210000000,
			LambdaBA:          250,
			LambdaDKG:         10000,
			NotaryParamAlpha:  70.5,
			NotaryParamBeta:   264,
			RoundLength:       3600,
			MinBlockInterval:  1000,
			FineValues: []*big.Int{
				new(big.Int).Mul(big.NewInt(1e18), big.NewInt(200)),
				new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1)),
				new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1e6)),
				new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1e6)),
				new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1e6)),
			},
		},
		Recovery: &RecoveryConfig{
			Contract:     common.HexToAddress("0xcb4bb8ae26b2ebe5a1e2e8d5236020f33ffb2294"),
			Timeout:      120,
			Confirmation: 5,
		},
	}

	// MainnetTrustedCheckpoint contains the light client trusted checkpoint for the main network.
	MainnetTrustedCheckpoint = &TrustedCheckpoint{
		Name:         "mainnet",
		SectionIndex: 227,
		SectionHead:  common.HexToHash("0xa2e0b25d72c2fc6e35a7f853cdacb193b4b4f95c606accf7f8fa8415283582c7"),
		CHTRoot:      common.HexToHash("0xf69bdd4053b95b61a27b106a0e86103d791edd8574950dc96aa351ab9b9f1aa0"),
		BloomRoot:    common.HexToHash("0xec1b454d4c6322c78ccedf76ac922a8698c3cac4d98748a84af4995b7bd3d744"),
	}

	// TestnetChainConfig contains the chain parameters to run a node on the Taiwan test network.
	TestnetChainConfig = &ChainConfig{
		ChainID:             big.NewInt(238),
		DMoment:             1554694200,
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        nil,
		DAOForkSupport:      true,
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		Dexcon: &DexconConfig{
			GenesisCRSText:    "In DEXON, we trust.",
			Owner:             common.HexToAddress("BF8C48A620bacc46907f9B89732D25E47A2D7Cf7"),
			MinStake:          new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1e6)),
			LockupPeriod:      86400 * 1000,
			MiningVelocity:    0.1875,
			NextHalvingSupply: new(big.Int).Mul(big.NewInt(1e18), big.NewInt(2.5e9)),
			LastHalvedAmount:  new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1.5e9)),
			MinGasPrice:       new(big.Int).Mul(big.NewInt(1e9), big.NewInt(24)),
			BlockGasLimit:     210000000,
			LambdaBA:          250,
			LambdaDKG:         10000,
			NotaryParamAlpha:  70.5,
			NotaryParamBeta:   264,
			RoundLength:       1200,
			MinBlockInterval:  1000,
			FineValues: []*big.Int{
				new(big.Int).Mul(big.NewInt(1e18), big.NewInt(200)),
				new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1)),
				new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1e6)),
				new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1e6)),
				new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1e6)),
			},
		},
		Recovery: &RecoveryConfig{
			Contract:     common.HexToAddress("0x4ebe3d13ab18b30d815711b7a33ef1226777b66d"),
			Timeout:      120,
			Confirmation: 5,
		},
	}

	// TaipeiChainConfig contains the chain parameters to run a node on the Taipei test network.
	TaipeiChainConfig = &ChainConfig{
		ChainID:             big.NewInt(239),
		DMoment:             1554388800,
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        nil,
		DAOForkSupport:      true,
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		Dexcon: &DexconConfig{
			GenesisCRSText:    "In DEXON, we trust.",
			Owner:             common.HexToAddress("BF8C48A620bacc46907f9B89732D25E47A2D7Cf7"),
			MinStake:          new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1e6)),
			LockupPeriod:      3600 * 2 * 1000,
			MiningVelocity:    0.1875,
			NextHalvingSupply: new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1.8e8)),
			LastHalvedAmount:  new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1.6e7)),
			MinGasPrice:       new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1)),
			BlockGasLimit:     21000 * 10000,
			LambdaBA:          250,
			LambdaDKG:         10000,
			NotaryParamAlpha:  70.5,
			NotaryParamBeta:   264,
			RoundLength:       1200,
			MinBlockInterval:  500,
			FineValues: []*big.Int{
				new(big.Int).Mul(big.NewInt(1e18), big.NewInt(200)),
				new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1)),
				new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1e6)),
				new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1e6)),
				new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1e6)),
			},
		},
		Recovery: &RecoveryConfig{
			Contract:     common.HexToAddress("0xac86ab80ab27007801f36f6622fbe0a9432291a2"),
			Timeout:      120,
			Confirmation: 1,
		},
	}

	// TestnetTrustedCheckpoint contains the light client trusted checkpoint for the Ropsten test network.
	TestnetTrustedCheckpoint = &TrustedCheckpoint{
		Name:         "testnet",
		SectionIndex: 161,
		SectionHead:  common.HexToHash("0x5378afa734e1feafb34bcca1534c4d96952b754579b96a4afb23d5301ecececc"),
		CHTRoot:      common.HexToHash("0x1cf2b071e7443a62914362486b613ff30f60cea0d9c268ed8c545f876a3ee60c"),
		BloomRoot:    common.HexToHash("0x5ac25c84bd18a9cbe878d4609a80220f57f85037a112644532412ba0d498a31b"),
	}

	// YilanChainConfig contains the chain parameters to run a node on the Yilan test network.
	YilanChainConfig = &ChainConfig{
		ChainID:             big.NewInt(240),
		DMoment:             1550802900,
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        nil,
		DAOForkSupport:      true,
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		Dexcon: &DexconConfig{
			GenesisCRSText:    "In DEXON, we trust, at Yilan",
			Owner:             common.HexToAddress("BF8C48A620bacc46907f9B89732D25E47A2D7Cf7"),
			MinStake:          new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1e6)),
			LockupPeriod:      86400 * 3 * 1000,
			MiningVelocity:    0.1875,
			NextHalvingSupply: new(big.Int).Mul(big.NewInt(1e18), big.NewInt(2e7)),
			LastHalvedAmount:  new(big.Int).Mul(big.NewInt(1e18), big.NewInt(4e6)),
			MinGasPrice:       new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1)),
			BlockGasLimit:     21000 * 5000,
			LambdaBA:          250,
			LambdaDKG:         10000,
			NotaryParamAlpha:  70.5,
			NotaryParamBeta:   264,
			RoundLength:       1200,
			MinBlockInterval:  500,
			FineValues: []*big.Int{
				new(big.Int).Mul(big.NewInt(1e18), big.NewInt(200)),
				new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1)),
				new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1e6)),
				new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1e6)),
				new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1e6)),
			},
		},
		Recovery: &RecoveryConfig{
			Contract:     common.HexToAddress("0x3828134ba7a0629fd52067b80fe696f400eb83dc"),
			Timeout:      120,
			Confirmation: 1,
		},
	}

	// AllEthashProtocolChanges contains every protocol change (EIPs) introduced
	// and accepted by the Ethereum core developers into the Ethash consensus.
	//
	// This configuration is intentionally not using keyed fields to force anyone
	// adding flags to the config to also have to set these fields.
	AllEthashProtocolChanges = &ChainConfig{big.NewInt(1337), 0, big.NewInt(0), nil, false, big.NewInt(0), common.Hash{}, big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), nil, new(EthashConfig), nil, nil, nil}

	// AllCliqueProtocolChanges contains every protocol change (EIPs) introduced
	// and accepted by the Ethereum core developers into the Clique consensus.
	//
	// This configuration is intentionally not using keyed fields to force anyone
	// adding flags to the config to also have to set these fields.
	AllCliqueProtocolChanges = &ChainConfig{big.NewInt(1337), 0, big.NewInt(0), nil, false, big.NewInt(0), common.Hash{}, big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), nil, nil, &CliqueConfig{Period: 0, Epoch: 30000}, nil, nil}

	AllDexconProtocolChanges = &ChainConfig{big.NewInt(1337), 0, big.NewInt(0), nil, false, big.NewInt(0), common.Hash{}, big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), nil, nil, nil, new(DexconConfig), new(RecoveryConfig)}

	TestChainConfig = &ChainConfig{big.NewInt(1), 0, big.NewInt(0), nil, false, big.NewInt(0), common.Hash{}, big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), nil, new(EthashConfig), nil, nil, nil}
	TestRules       = TestChainConfig.Rules(new(big.Int))

	// Ethereum MainnetChainConfig is the chain parameters to run a node on the main network.
	EthereumMainnetChainConfig = &ChainConfig{
		ChainID:             big.NewInt(1),
		DMoment:             0,
		HomesteadBlock:      big.NewInt(1150000),
		DAOForkBlock:        big.NewInt(1920000),
		DAOForkSupport:      true,
		EIP150Block:         big.NewInt(2463000),
		EIP150Hash:          common.HexToHash("0x2086799aeebeae135c246c65021c82b4e15a2c451340993aacfd2751886514f0"),
		EIP155Block:         big.NewInt(2675000),
		EIP158Block:         big.NewInt(2675000),
		ByzantiumBlock:      big.NewInt(4370000),
		ConstantinopleBlock: nil,
		Ethash:              new(EthashConfig),
	}

	// Ethereum TestnetChainConfig contains the chain parameters to run a node on the Ropsten test network.
	EthereumTestnetChainConfig = &ChainConfig{
		ChainID:             big.NewInt(3),
		DMoment:             0,
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        nil,
		DAOForkSupport:      true,
		EIP150Block:         big.NewInt(0),
		EIP150Hash:          common.HexToHash("0x41941023680923e0fe4d74a34bdac8141f2540e3ae90623718e47d66d1ca4a2d"),
		EIP155Block:         big.NewInt(10),
		EIP158Block:         big.NewInt(10),
		ByzantiumBlock:      big.NewInt(1700000),
		ConstantinopleBlock: big.NewInt(4230000),
		Ethash:              new(EthashConfig),
	}
)

// TrustedCheckpoint represents a set of post-processed trie roots (CHT and
// BloomTrie) associated with the appropriate section index and head hash. It is
// used to start light syncing from this checkpoint and avoid downloading the
// entire header chain while still being able to securely access old headers/logs.
type TrustedCheckpoint struct {
	Name         string      `json:"-"`
	SectionIndex uint64      `json:"sectionIndex"`
	SectionHead  common.Hash `json:"sectionHead"`
	CHTRoot      common.Hash `json:"chtRoot"`
	BloomRoot    common.Hash `json:"bloomRoot"`
}

// ChainConfig is the core config which determines the blockchain settings.
//
// ChainConfig is stored in the database on a per block basis. This means
// that any network, identified by its genesis block, can have its own
// set of configuration options.
type ChainConfig struct {
	ChainID *big.Int `json:"chainId"` // chainId identifies the current chain and is used for replay protection
	DMoment uint64   `json:"dMoment"` // dMoment which indicate the starting time of the network

	HomesteadBlock *big.Int `json:"homesteadBlock,omitempty"` // Homestead switch block (nil = no fork, 0 = already homestead)

	DAOForkBlock   *big.Int `json:"daoForkBlock,omitempty"`   // TheDAO hard-fork switch block (nil = no fork)
	DAOForkSupport bool     `json:"daoForkSupport,omitempty"` // Whether the nodes supports or opposes the DAO hard-fork

	// EIP150 implements the Gas price changes (https://github.com/ethereum/EIPs/issues/150)
	EIP150Block *big.Int    `json:"eip150Block,omitempty"` // EIP150 HF block (nil = no fork)
	EIP150Hash  common.Hash `json:"eip150Hash,omitempty"`  // EIP150 HF hash (needed for header only clients as only gas pricing changed)

	EIP155Block *big.Int `json:"eip155Block,omitempty"` // EIP155 HF block
	EIP158Block *big.Int `json:"eip158Block,omitempty"` // EIP158 HF block

	ByzantiumBlock      *big.Int `json:"byzantiumBlock,omitempty"`      // Byzantium switch block (nil = no fork, 0 = already on byzantium)
	ConstantinopleBlock *big.Int `json:"constantinopleBlock,omitempty"` // Constantinople switch block (nil = no fork, 0 = already activated)
	PetersburgBlock     *big.Int `json:"petersburgBlock,omitempty"`     // Petersburg switch block (nil = same as Constantinople)
	EWASMBlock          *big.Int `json:"ewasmBlock,omitempty"`          // EWASM switch block (nil = no fork, 0 = already activated)

	// Various consensus engines
	Ethash *EthashConfig `json:"ethash,omitempty"`
	Clique *CliqueConfig `json:"clique,omitempty"`
	Dexcon *DexconConfig `json:"dexcon,omitempty"`

	// Dexcon Recovery
	Recovery *RecoveryConfig `json:"recovery,omitempty"`
}

// EthashConfig is the consensus engine configs for proof-of-work based sealing.
type EthashConfig struct{}

// String implements the stringer interface, returning the consensus engine details.
func (c *EthashConfig) String() string {
	return "ethash"
}

// CliqueConfig is the consensus engine configs for proof-of-authority based sealing.
type CliqueConfig struct {
	Period uint64 `json:"period"` // Number of seconds between blocks to enforce
	Epoch  uint64 `json:"epoch"`  // Epoch length to reset votes and checkpoint
}

// String implements the stringer interface, returning the consensus engine details.
func (c *CliqueConfig) String() string {
	return "clique"
}

//go:generate gencodec -type DexconConfig -field-override dexconConfigSpecMarshaling -out gen_dexcon_config.go

// DexconConfig is the consensus engine configs for DEXON consensus.
type DexconConfig struct {
	GenesisCRSText    string         `json:"genesisCRSText"`
	Owner             common.Address `json:"owner"`
	MinStake          *big.Int       `json:"minStake"`
	LockupPeriod      uint64         `json:"lockupPeriod"`
	MiningVelocity    float32        `json:"miningVelocity"`
	NextHalvingSupply *big.Int       `json:"nextHalvingSupply"`
	LastHalvedAmount  *big.Int       `json:"lastHalvedAmount"`
	MinGasPrice       *big.Int       `json:"minGasPrice"`
	BlockGasLimit     uint64         `json:"blockGasLimit"`
	LambdaBA          uint64         `json:"lambdaBA"`
	LambdaDKG         uint64         `json:"lambdaDKG"`
	NotaryParamAlpha  float32        `json:"notaryParamAlpha"`
	NotaryParamBeta   float32        `json:"notaryParamBeta"`
	RoundLength       uint64         `json:"roundLength"`
	MinBlockInterval  uint64         `json:"minBlockInterval"`
	FineValues        []*big.Int     `json:"fineValues"`
}

type dexconConfigSpecMarshaling struct {
	MinStake          *math.HexOrDecimal256
	NextHalvingSupply *math.HexOrDecimal256
	LastHalvedAmount  *math.HexOrDecimal256
	MinGasPrice       *math.HexOrDecimal256
	FineValues        []*math.HexOrDecimal256
}

// String implements the stringer interface, returning the consensus engine details.
func (d *DexconConfig) String() string {
	return fmt.Sprintf("{GenesisCRSText: %v Owner: %v MinStake: %v LockupPeriod: %v MiningVelocity: %v NextHalvingSupply: %v LastHalvedAmount: %v MinGasPrice: %v BlockGasLimit: %v LambdaBA: %v LambdaDKG: %v NotaryParamAlpha: %v NotaryParamBeta: %v RoundLength: %v MinBlockInterval: %v FineValues: %v}",
		d.GenesisCRSText,
		d.Owner,
		d.MinStake,
		d.LockupPeriod,
		d.MiningVelocity,
		d.NextHalvingSupply,
		d.LastHalvedAmount,
		d.MinGasPrice,
		d.BlockGasLimit,
		d.LambdaBA,
		d.LambdaDKG,
		d.NotaryParamAlpha,
		d.NotaryParamBeta,
		d.RoundLength,
		d.MinBlockInterval,
		d.FineValues,
	)
}

type RecoveryConfig struct {
	Contract     common.Address `json:"contract"`
	Timeout      int            `json:"timeout"`
	Confirmation int            `json:"confirmation"`
}

// String implements the fmt.Stringer interface.
func (c *ChainConfig) String() string {
	var engine interface{}
	switch {
	case c.Ethash != nil:
		engine = c.Ethash
	case c.Clique != nil:
		engine = c.Clique
	default:
		engine = "unknown"
	}
	return fmt.Sprintf("{ChainID: %v Homestead: %v DAO: %v DAOSupport: %v EIP150: %v EIP155: %v EIP158: %v Byzantium: %v Constantinople: %v  ConstantinopleFix: %v Engine: %v}",
		c.ChainID,
		c.HomesteadBlock,
		c.DAOForkBlock,
		c.DAOForkSupport,
		c.EIP150Block,
		c.EIP155Block,
		c.EIP158Block,
		c.ByzantiumBlock,
		c.ConstantinopleBlock,
		c.PetersburgBlock,
		engine,
	)
}

// IsHomestead returns whether num is either equal to the homestead block or greater.
func (c *ChainConfig) IsHomestead(num *big.Int) bool {
	return isForked(c.HomesteadBlock, num)
}

// IsDAOFork returns whether num is either equal to the DAO fork block or greater.
func (c *ChainConfig) IsDAOFork(num *big.Int) bool {
	return isForked(c.DAOForkBlock, num)
}

// IsEIP150 returns whether num is either equal to the EIP150 fork block or greater.
func (c *ChainConfig) IsEIP150(num *big.Int) bool {
	return isForked(c.EIP150Block, num)
}

// IsEIP155 returns whether num is either equal to the EIP155 fork block or greater.
func (c *ChainConfig) IsEIP155(num *big.Int) bool {
	return isForked(c.EIP155Block, num)
}

// IsEIP158 returns whether num is either equal to the EIP158 fork block or greater.
func (c *ChainConfig) IsEIP158(num *big.Int) bool {
	return isForked(c.EIP158Block, num)
}

// IsByzantium returns whether num is either equal to the Byzantium fork block or greater.
func (c *ChainConfig) IsByzantium(num *big.Int) bool {
	return isForked(c.ByzantiumBlock, num)
}

// IsConstantinople returns whether num is either equal to the Constantinople fork block or greater.
func (c *ChainConfig) IsConstantinople(num *big.Int) bool {
	return isForked(c.ConstantinopleBlock, num)
}

// IsPetersburg returns whether num is either
// - equal to or greater than the PetersburgBlock fork block,
// - OR is nil, and Constantinople is active
func (c *ChainConfig) IsPetersburg(num *big.Int) bool {
	return isForked(c.PetersburgBlock, num) || c.PetersburgBlock == nil && isForked(c.ConstantinopleBlock, num)
}

// IsEWASM returns whether num represents a block number after the EWASM fork
func (c *ChainConfig) IsEWASM(num *big.Int) bool {
	return isForked(c.EWASMBlock, num)
}

// GasTable returns the gas table corresponding to the current phase (homestead or homestead reprice).
//
// The returned GasTable's fields shouldn't, under any circumstances, be changed.
func (c *ChainConfig) GasTable(num *big.Int) GasTable {
	if num == nil {
		return GasTableHomestead
	}
	switch {
	case c.IsConstantinople(num):
		return GasTableConstantinople
	case c.IsEIP158(num):
		return GasTableEIP158
	case c.IsEIP150(num):
		return GasTableEIP150
	default:
		return GasTableHomestead
	}
}

// CheckCompatible checks whether scheduled fork transitions have been imported
// with a mismatching chain configuration.
func (c *ChainConfig) CheckCompatible(newcfg *ChainConfig, height uint64) *ConfigCompatError {
	bhead := new(big.Int).SetUint64(height)

	// Iterate checkCompatible to find the lowest conflict.
	var lasterr *ConfigCompatError
	for {
		err := c.checkCompatible(newcfg, bhead)
		if err == nil || (lasterr != nil && err.RewindTo == lasterr.RewindTo) {
			break
		}
		lasterr = err
		bhead.SetUint64(err.RewindTo)
	}
	return lasterr
}

func (c *ChainConfig) checkCompatible(newcfg *ChainConfig, head *big.Int) *ConfigCompatError {
	if isForkIncompatible(c.HomesteadBlock, newcfg.HomesteadBlock, head) {
		return newCompatError("Homestead fork block", c.HomesteadBlock, newcfg.HomesteadBlock)
	}
	if isForkIncompatible(c.DAOForkBlock, newcfg.DAOForkBlock, head) {
		return newCompatError("DAO fork block", c.DAOForkBlock, newcfg.DAOForkBlock)
	}
	if c.IsDAOFork(head) && c.DAOForkSupport != newcfg.DAOForkSupport {
		return newCompatError("DAO fork support flag", c.DAOForkBlock, newcfg.DAOForkBlock)
	}
	if isForkIncompatible(c.EIP150Block, newcfg.EIP150Block, head) {
		return newCompatError("EIP150 fork block", c.EIP150Block, newcfg.EIP150Block)
	}
	if isForkIncompatible(c.EIP155Block, newcfg.EIP155Block, head) {
		return newCompatError("EIP155 fork block", c.EIP155Block, newcfg.EIP155Block)
	}
	if isForkIncompatible(c.EIP158Block, newcfg.EIP158Block, head) {
		return newCompatError("EIP158 fork block", c.EIP158Block, newcfg.EIP158Block)
	}
	if c.IsEIP158(head) && !configNumEqual(c.ChainID, newcfg.ChainID) {
		return newCompatError("EIP158 chain ID", c.EIP158Block, newcfg.EIP158Block)
	}
	if isForkIncompatible(c.ByzantiumBlock, newcfg.ByzantiumBlock, head) {
		return newCompatError("Byzantium fork block", c.ByzantiumBlock, newcfg.ByzantiumBlock)
	}
	if isForkIncompatible(c.ConstantinopleBlock, newcfg.ConstantinopleBlock, head) {
		return newCompatError("Constantinople fork block", c.ConstantinopleBlock, newcfg.ConstantinopleBlock)
	}
	if isForkIncompatible(c.PetersburgBlock, newcfg.PetersburgBlock, head) {
		return newCompatError("ConstantinopleFix fork block", c.PetersburgBlock, newcfg.PetersburgBlock)
	}
	if isForkIncompatible(c.EWASMBlock, newcfg.EWASMBlock, head) {
		return newCompatError("ewasm fork block", c.EWASMBlock, newcfg.EWASMBlock)
	}
	return nil
}

// isForkIncompatible returns true if a fork scheduled at s1 cannot be rescheduled to
// block s2 because head is already past the fork.
func isForkIncompatible(s1, s2, head *big.Int) bool {
	return (isForked(s1, head) || isForked(s2, head)) && !configNumEqual(s1, s2)
}

// isForked returns whether a fork scheduled at block s is active at the given head block.
func isForked(s, head *big.Int) bool {
	if s == nil || head == nil {
		return false
	}

	return s.Cmp(head) <= 0
}

func configNumEqual(x, y *big.Int) bool {
	if x == nil {
		return y == nil
	}
	if y == nil {
		return x == nil
	}
	return x.Cmp(y) == 0
}

// ConfigCompatError is raised if the locally-stored blockchain is initialised with a
// ChainConfig that would alter the past.
type ConfigCompatError struct {
	What string
	// block numbers of the stored and new configurations
	StoredConfig, NewConfig *big.Int
	// the block number to which the local chain must be rewound to correct the error
	RewindTo uint64
}

func newCompatError(what string, storedblock, newblock *big.Int) *ConfigCompatError {
	var rew *big.Int
	switch {
	case storedblock == nil:
		rew = newblock
	case newblock == nil || storedblock.Cmp(newblock) < 0:
		rew = storedblock
	default:
		rew = newblock
	}
	err := &ConfigCompatError{what, storedblock, newblock, 0}
	if rew != nil && rew.Sign() > 0 {
		err.RewindTo = rew.Uint64() - 1
	}
	return err
}

func (err *ConfigCompatError) Error() string {
	return fmt.Sprintf("mismatching %s in database (have %d, want %d, rewindto %d)", err.What, err.StoredConfig, err.NewConfig, err.RewindTo)
}

// Rules wraps ChainConfig and is merely syntactic sugar or can be used for functions
// that do not have or require information about the block.
//
// Rules is a one time interface meaning that it shouldn't be used in between transition
// phases.
type Rules struct {
	ChainID                                     *big.Int
	IsHomestead, IsEIP150, IsEIP155, IsEIP158   bool
	IsByzantium, IsConstantinople, IsPetersburg bool
}

// Rules ensures c's ChainID is not nil.
func (c *ChainConfig) Rules(num *big.Int) Rules {
	chainID := c.ChainID
	if chainID == nil {
		chainID = new(big.Int)
	}
	return Rules{
		ChainID:          new(big.Int).Set(chainID),
		IsHomestead:      c.IsHomestead(num),
		IsEIP150:         c.IsEIP150(num),
		IsEIP155:         c.IsEIP155(num),
		IsEIP158:         c.IsEIP158(num),
		IsByzantium:      c.IsByzantium(num),
		IsConstantinople: c.IsConstantinople(num),
		IsPetersburg:     c.IsPetersburg(num),
	}
}
