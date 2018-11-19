package dex

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"math/big"
	"time"

	coreCommon "github.com/dexon-foundation/dexon-consensus/common"
	dexCore "github.com/dexon-foundation/dexon-consensus/core"
	coreCrypto "github.com/dexon-foundation/dexon-consensus/core/crypto"
	coreEcdsa "github.com/dexon-foundation/dexon-consensus/core/crypto/ecdsa"
	coreTypes "github.com/dexon-foundation/dexon-consensus/core/types"
	dkgTypes "github.com/dexon-foundation/dexon-consensus/core/types/dkg"

	"github.com/dexon-foundation/dexon/common"
	"github.com/dexon-foundation/dexon/core/types"
	"github.com/dexon-foundation/dexon/core/vm"
	"github.com/dexon-foundation/dexon/crypto"
	"github.com/dexon-foundation/dexon/log"
	"github.com/dexon-foundation/dexon/params"
	"github.com/dexon-foundation/dexon/rlp"
	"github.com/dexon-foundation/dexon/rpc"
)

type DexconGovernance struct {
	b            *DexAPIBackend
	chainConfig  *params.ChainConfig
	privateKey   *ecdsa.PrivateKey
	address      common.Address
	nodeSetCache *dexCore.NodeSetCache
}

// NewDexconGovernance retruns a governance implementation of the DEXON
// consensus governance interface.
func NewDexconGovernance(backend *DexAPIBackend, chainConfig *params.ChainConfig,
	privKey *ecdsa.PrivateKey) *DexconGovernance {
	g := &DexconGovernance{
		b:           backend,
		chainConfig: chainConfig,
		privateKey:  privKey,
		address:     crypto.PubkeyToAddress(privKey.PublicKey),
	}
	g.nodeSetCache = dexCore.NewNodeSetCache(g)
	return g
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
	if round < dexCore.ConfigRoundShift {
		round = 0
	} else {
		round -= dexCore.ConfigRoundShift
	}
	ctx := context.Background()
	blockHeight, err := d.getRoundHeight(ctx, round)
	if err != nil {
		return nil
	}

	state, _, err := d.b.StateAndHeaderByNumber(ctx, rpc.BlockNumber(blockHeight))
	if state == nil || err != nil {
		log.Error("Failed to retrieve governance state", "round", round, "height", blockHeight)
		return nil
	}
	return &vm.GovernanceStateHelper{state}
}

// DexconConfiguration return raw config in state.
func (d *DexconGovernance) DexconConfiguration(round uint64) *params.DexconConfig {
	s := d.getGovStateAtRound(round)
	return s.Configuration()
}

// Configuration returns the system configuration for consensus core to use.
func (d *DexconGovernance) Configuration(round uint64) *coreTypes.Config {
	s := d.getGovStateAtRound(round)
	c := s.Configuration()

	return &coreTypes.Config{
		NumChains:        c.NumChains,
		LambdaBA:         time.Duration(c.LambdaBA) * time.Millisecond,
		LambdaDKG:        time.Duration(c.LambdaDKG) * time.Millisecond,
		K:                int(c.K),
		PhiRatio:         c.PhiRatio,
		NotarySetSize:    c.NotarySetSize,
		DKGSetSize:       c.DKGSetSize,
		RoundInterval:    time.Duration(c.RoundInterval) * time.Millisecond,
		MinBlockInterval: time.Duration(c.MinBlockInterval) * time.Millisecond,
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

	// Increase gasPrice to 10 times of suggested gas price to make sure it will
	// be included in time.
	gasPrice = new(big.Int).Mul(gasPrice, big.NewInt(10))

	tx := types.NewTransaction(
		nonce,
		vm.GovernanceContractAddress,
		big.NewInt(0),
		uint64(10000000),
		gasPrice,
		data)

	signer := types.NewEIP155Signer(d.chainConfig.ChainID)

	tx, err = types.SignTx(tx, signer, d.privateKey)
	if err != nil {
		return err
	}

	log.Info("Send governance transaction", "fullhash", tx.Hash().Hex(), "nonce", nonce)

	return d.b.SendTx(ctx, tx)
}

// CRS returns the CRS for a given round.
func (d *DexconGovernance) CRS(round uint64) coreCommon.Hash {
	s := d.getGovState()
	return coreCommon.Hash(s.CRS(big.NewInt(int64(round))))
}

func (d *DexconGovernance) LenCRS() uint64 {
	s := d.getGovState()
	return s.LenCRS().Uint64()
}

// ProposeCRS send proposals of a new CRS
func (d *DexconGovernance) ProposeCRS(round uint64, signedCRS []byte) {
	method := vm.GovernanceContractName2Method["proposeCRS"]

	res, err := method.Inputs.Pack(big.NewInt(int64(round)), signedCRS)
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

// NodeSet returns the current node set.
func (d *DexconGovernance) NodeSet(round uint64) []coreCrypto.PublicKey {
	s := d.getGovStateAtRound(round)
	var pks []coreCrypto.PublicKey

	for _, n := range s.QualifiedNodes() {
		pk, err := coreEcdsa.NewPublicKeyFromByteSlice(n.PublicKey)
		if err != nil {
			panic(err)
		}
		pks = append(pks, pk)
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
func (d *DexconGovernance) AddDKGComplaint(round uint64, complaint *dkgTypes.Complaint) {
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
func (d *DexconGovernance) DKGComplaints(round uint64) []*dkgTypes.Complaint {
	s := d.getGovState()
	var dkgComplaints []*dkgTypes.Complaint
	for _, pk := range s.DKGComplaints(big.NewInt(int64(round))) {
		x := new(dkgTypes.Complaint)
		if err := rlp.DecodeBytes(pk, x); err != nil {
			panic(err)
		}
		dkgComplaints = append(dkgComplaints, x)
	}
	return dkgComplaints
}

// AddDKGMasterPublicKey adds a DKGMasterPublicKey.
func (d *DexconGovernance) AddDKGMasterPublicKey(round uint64, masterPublicKey *dkgTypes.MasterPublicKey) {
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
func (d *DexconGovernance) DKGMasterPublicKeys(round uint64) []*dkgTypes.MasterPublicKey {
	s := d.getGovState()
	var dkgMasterPKs []*dkgTypes.MasterPublicKey
	for _, pk := range s.DKGMasterPublicKeys(big.NewInt(int64(round))) {
		x := new(dkgTypes.MasterPublicKey)
		if err := rlp.DecodeBytes(pk, x); err != nil {
			panic(err)
		}
		dkgMasterPKs = append(dkgMasterPKs, x)
	}
	return dkgMasterPKs
}

// AddDKGFinalize adds a DKG finalize message.
func (d *DexconGovernance) AddDKGFinalize(round uint64, final *dkgTypes.Finalize) {
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
	s := d.getGovState()
	threshold := 2*s.DKGSetSize().Uint64()/3 + 1
	count := s.DKGFinalizedsCount(big.NewInt(int64(round))).Uint64()
	return count >= threshold
}

func (d *DexconGovernance) GetNumChains(round uint64) uint32 {
	return d.Configuration(round).NumChains
}

func (d *DexconGovernance) NotarySet(round uint64, chainID uint32) (map[string]struct{}, error) {
	notarySet, err := d.nodeSetCache.GetNotarySet(round, chainID)
	if err != nil {
		return nil, err
	}

	r := make(map[string]struct{}, len(notarySet))
	for id := range notarySet {
		if key, exists := d.nodeSetCache.GetPublicKey(id); exists {
			r[hex.EncodeToString(key.Bytes())] = struct{}{}
		}
	}
	return r, nil
}

func (d *DexconGovernance) DKGSet(round uint64) (map[string]struct{}, error) {
	dkgSet, err := d.nodeSetCache.GetDKGSet(round)
	if err != nil {
		return nil, err
	}

	r := make(map[string]struct{}, len(dkgSet))
	for id := range dkgSet {
		if key, exists := d.nodeSetCache.GetPublicKey(id); exists {
			r[hex.EncodeToString(key.Bytes())] = struct{}{}
		}
	}
	return r, nil
}
