package dex

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"time"

	coreCommon "github.com/dexon-foundation/dexon-consensus-core/common"
	coreCrypto "github.com/dexon-foundation/dexon-consensus-core/core/crypto"
	coreEcdsa "github.com/dexon-foundation/dexon-consensus-core/core/crypto/ecdsa"
	coreTypes "github.com/dexon-foundation/dexon-consensus-core/core/types"

	"github.com/dexon-foundation/dexon/common"
	"github.com/dexon-foundation/dexon/core"
	"github.com/dexon-foundation/dexon/core/types"
	"github.com/dexon-foundation/dexon/core/vm"
	"github.com/dexon-foundation/dexon/crypto"
	"github.com/dexon-foundation/dexon/event"
	"github.com/dexon-foundation/dexon/log"
	"github.com/dexon-foundation/dexon/params"
	"github.com/dexon-foundation/dexon/rlp"
	"github.com/dexon-foundation/dexon/rpc"
)

type DexconGovernance struct {
	b           *DexAPIBackend
	chainConfig *params.ChainConfig
	privateKey  *ecdsa.PrivateKey
	address     common.Address
}

// NewDexconGovernance retruns a governance implementation of the DEXON
// consensus governance interface.
func NewDexconGovernance(backend *DexAPIBackend, chainConfig *params.ChainConfig,
	privKey *ecdsa.PrivateKey) *DexconGovernance {
	return &DexconGovernance{
		b:           backend,
		chainConfig: chainConfig,
		privateKey:  privKey,
		address:     crypto.PubkeyToAddress(privKey.PublicKey),
	}
}

func (d *DexconGovernance) getRoundHeight(ctx context.Context, round uint64) (uint64, error) {
	state, _, err := d.b.StateAndHeaderByNumber(ctx, rpc.LatestBlockNumber)
	if state == nil || err != nil {
		return 0, err
	}
	s := vm.GovernanceStateHelper{state}
	return s.RoundHeight(big.NewInt(int64(round))).Uint64(), nil
}

func (d *DexconGovernance) getGovState() *vm.GovernanceStateHelper {
	ctx := context.Background()
	state, _, err := d.b.StateAndHeaderByNumber(ctx, rpc.LatestBlockNumber)
	if state == nil || err != nil {
		return nil
	}

	return &vm.GovernanceStateHelper{state}
}

func (d *DexconGovernance) getGovStateAtRound(round uint64) *vm.GovernanceStateHelper {
	ctx := context.Background()
	blockHeight, err := d.getRoundHeight(ctx, round)
	if err != nil {
		return nil
	}

	state, _, err := d.b.StateAndHeaderByNumber(ctx, rpc.BlockNumber(blockHeight))
	if state == nil || err != nil {
		return nil
	}

	return &vm.GovernanceStateHelper{state}
}

// Configuration return the total ordering K constant.
func (d *DexconGovernance) Configuration(round uint64) *coreTypes.Config {
	s := d.getGovStateAtRound(round)
	c := s.Configuration()

	return &coreTypes.Config{
		NumChains:        c.NumChains,
		LambdaBA:         time.Duration(c.LambdaBA) * time.Millisecond,
		LambdaDKG:        time.Duration(c.LambdaDKG) * time.Millisecond,
		K:                c.K,
		PhiRatio:         c.PhiRatio,
		NotarySetSize:    c.NotarySetSize,
		DKGSetSize:       c.DKGSetSize,
		RoundInterval:    time.Duration(c.RoundInterval) * time.Millisecond,
		MinBlockInterval: time.Duration(c.MinBlockInterval) * time.Millisecond,
		MaxBlockInterval: time.Duration(c.MaxBlockInterval) * time.Millisecond,
	}
}

func (d *DexconGovernance) sendGovTx(ctx context.Context, data []byte) error {
	gasPrice, err := d.b.SuggestPrice(ctx)
	if err != nil {
		return err
	}

	nonce, err := d.b.GetPoolNonce(ctx, d.address)
	if err != nil {
		return err
	}

	tx := types.NewTransaction(
		nonce,
		vm.GovernanceContractAddress,
		big.NewInt(0),
		uint64(200000),
		gasPrice,
		data)

	signer := types.NewEIP155Signer(d.chainConfig.ChainID)

	tx, err = types.SignTx(tx, signer, d.privateKey)
	if err != nil {
		return err
	}
	return d.b.SendTx(ctx, tx)
}

// CRS returns the CRS for a given round.
func (d *DexconGovernance) CRS(round uint64) coreCommon.Hash {
	s := d.getGovStateAtRound(round)
	return coreCommon.Hash(s.CRS(big.NewInt(int64(round))))
}

// ProposeCRS send proposals of a new CRS
func (d *DexconGovernance) ProposeCRS(signedCRS []byte) {
	method := vm.GovernanceContractName2Method["proposeCRS"]

	res, err := method.Inputs.Pack(signedCRS)
	if err != nil {
		log.Error("failed to pack proposeCRS input", "err", err)
		return
	}

	data := append(method.Id(), res...)
	err = d.sendGovTx(context.Background(), data)
	if err != nil {
		log.Error("failed to send proposeCRS tx", "err", err)
	}
}

// NodeSet returns the current notary set.
func (d *DexconGovernance) NodeSet(round uint64) []coreCrypto.PublicKey {
	s := d.getGovStateAtRound(round)
	var pks []coreCrypto.PublicKey

	for _, n := range s.Nodes() {
		pks = append(pks, coreEcdsa.NewPublicKeyFromByteSlice(n.PublicKey))
	}
	return pks
}

// NotifyRoundHeight register the mapping between round and height.
func (d *DexconGovernance) NotifyRoundHeight(targetRound, consensusHeight uint64) {
	method := vm.GovernanceContractName2Method["snapshotRound"]

	res, err := method.Inputs.Pack(
		big.NewInt(int64(targetRound)), big.NewInt(int64(consensusHeight)))
	if err != nil {
		log.Error("failed to pack snapshotRound input", "err", err)
		return
	}

	data := append(method.Id(), res...)
	err = d.sendGovTx(context.Background(), data)
	if err != nil {
		log.Error("failed to send snapshotRound tx", "err", err)
	}
}

// AddDKGComplaint adds a DKGComplaint.
func (d *DexconGovernance) AddDKGComplaint(round uint64, complaint *coreTypes.DKGComplaint) {
	method := vm.GovernanceContractName2Method["addDKGComplaint"]

	encoded, err := rlp.EncodeToBytes(complaint)
	if err != nil {
		log.Error("failed to RLP encode complaint to bytes", "err", err)
		return
	}

	res, err := method.Inputs.Pack(big.NewInt(int64(round)), encoded)
	if err != nil {
		log.Error("failed to pack addDKGComplaint input", "err", err)
		return
	}

	data := append(method.Id(), res...)
	err = d.sendGovTx(context.Background(), data)
	if err != nil {
		log.Error("failed to send addDKGComplaint tx", "err", err)
	}
}

// DKGComplaints gets all the DKGComplaints of round.
func (d *DexconGovernance) DKGComplaints(round uint64) []*coreTypes.DKGComplaint {
	s := d.getGovState()
	var dkgComplaints []*coreTypes.DKGComplaint
	for _, pk := range s.DKGMasterPublicKeys(big.NewInt(int64(round))) {
		x := new(coreTypes.DKGComplaint)
		if err := rlp.DecodeBytes(pk, x); err != nil {
			panic(err)
		}
		dkgComplaints = append(dkgComplaints, x)
	}
	return dkgComplaints
}

// AddDKGMasterPublicKey adds a DKGMasterPublicKey.
func (d *DexconGovernance) AddDKGMasterPublicKey(round uint64, masterPublicKey *coreTypes.DKGMasterPublicKey) {
	method := vm.GovernanceContractName2Method["addDKGMasterPublicKey"]

	encoded, err := rlp.EncodeToBytes(masterPublicKey)
	if err != nil {
		log.Error("failed to RLP encode mpk to bytes", "err", err)
		return
	}

	res, err := method.Inputs.Pack(big.NewInt(int64(round)), encoded)
	if err != nil {
		log.Error("failed to pack addDKGMasterPublicKey input", "err", err)
		return
	}

	data := append(method.Id(), res...)
	err = d.sendGovTx(context.Background(), data)
	if err != nil {
		log.Error("failed to send addDKGMasterPublicKey tx", "err", err)
	}
}

// DKGMasterPublicKeys gets all the DKGMasterPublicKey of round.
func (d *DexconGovernance) DKGMasterPublicKeys(round uint64) []*coreTypes.DKGMasterPublicKey {
	s := d.getGovState()
	var dkgMasterPKs []*coreTypes.DKGMasterPublicKey
	for _, pk := range s.DKGMasterPublicKeys(big.NewInt(int64(round))) {
		x := new(coreTypes.DKGMasterPublicKey)
		if err := rlp.DecodeBytes(pk, x); err != nil {
			panic(err)
		}
		dkgMasterPKs = append(dkgMasterPKs, x)
	}
	return dkgMasterPKs
}

// AddDKGFinalize adds a DKG finalize message.
func (d *DexconGovernance) AddDKGFinalize(round uint64, final *coreTypes.DKGFinalize) {
	method := vm.GovernanceContractName2Method["addDKGFinalize"]

	encoded, err := rlp.EncodeToBytes(final)
	if err != nil {
		log.Error("failed to RLP encode finalize to bytes", "err", err)
		return
	}

	res, err := method.Inputs.Pack(big.NewInt(int64(round)), encoded)
	if err != nil {
		log.Error("failed to pack addDKGFinalize input", "err", err)
		return
	}

	data := append(method.Id(), res...)
	err = d.sendGovTx(context.Background(), data)
	if err != nil {
		log.Error("failed to send addDKGFinalize tx", "err", err)
	}
}

// IsDKGFinal checks if DKG is final.
func (d *DexconGovernance) IsDKGFinal(round uint64) bool {
	s := d.getGovStateAtRound(round)
	threshold := 2*s.DKGSetSize().Uint64()/3 + 1
	count := s.DKGFinalizedsCount(big.NewInt(int64(round))).Uint64()
	return count >= threshold
}

// TODO(sonic): finish these
func (d *DexconGovernance) GetChainNum(uint64) uint32 {
	return 3
}

func (d *DexconGovernance) GetNotarySet(uint32, uint64) map[string]struct{} {
	return nil
}

func (d *DexconGovernance) GetDKGSet(uint64) map[string]struct{} {
	return nil
}

func (d *DexconGovernance) SubscribeNewCRSEvent(ch chan core.NewCRSEvent) event.Subscription {
	return nil
}
