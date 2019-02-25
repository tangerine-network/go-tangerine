// Copyright 2018 The dexon-consensus Authors
// This file is part of the dexon-consensus library.
//
// The dexon-consensus library is free software: you can redistribute it
// and/or modify it under the terms of the GNU Lesser General Public License as
// published by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// The dexon-consensus library is distributed in the hope that it will be
// useful, but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Lesser
// General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the dexon-consensus library. If not, see
// <http://www.gnu.org/licenses/>.

package dex

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"math/big"

	coreCommon "github.com/dexon-foundation/dexon-consensus/common"
	dexCore "github.com/dexon-foundation/dexon-consensus/core"
	coreCrypto "github.com/dexon-foundation/dexon-consensus/core/crypto"
	coreEcdsa "github.com/dexon-foundation/dexon-consensus/core/crypto/ecdsa"
	coreTypes "github.com/dexon-foundation/dexon-consensus/core/types"
	dkgTypes "github.com/dexon-foundation/dexon-consensus/core/types/dkg"

	"github.com/dexon-foundation/dexon/common"
	"github.com/dexon-foundation/dexon/core"
	"github.com/dexon-foundation/dexon/core/types"
	"github.com/dexon-foundation/dexon/core/vm"
	"github.com/dexon-foundation/dexon/crypto"
	"github.com/dexon-foundation/dexon/log"
	"github.com/dexon-foundation/dexon/params"
)

type DexconGovernance struct {
	*core.Governance

	b            *DexAPIBackend
	chainConfig  *params.ChainConfig
	privateKey   *ecdsa.PrivateKey
	address      common.Address
	nodeSetCache *dexCore.NodeSetCache
}

// NewDexconGovernance returns a governance implementation of the DEXON
// consensus governance interface.
func NewDexconGovernance(backend *DexAPIBackend, chainConfig *params.ChainConfig,
	privKey *ecdsa.PrivateKey) *DexconGovernance {
	g := &DexconGovernance{
		Governance: core.NewGovernance(
			core.NewGovernanceStateDB(backend.dex.BlockChain())),
		b:           backend,
		chainConfig: chainConfig,
		privateKey:  privKey,
		address:     crypto.PubkeyToAddress(privKey.PublicKey),
	}
	g.nodeSetCache = dexCore.NewNodeSetCache(g)
	return g
}

// DexconConfiguration return raw config in state.
func (d *DexconGovernance) DexconConfiguration(round uint64) *params.DexconConfig {
	return d.GetStateForConfigAtRound(round).Configuration()
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
	if round <= dexCore.DKGDelayRound {
		s := d.GetStateAtRound(0)
		crs := s.CRS()
		for i := uint64(0); i < round; i++ {
			crs = crypto.Keccak256Hash(crs[:])
		}
		return coreCommon.Hash(crs)
	}
	if round > d.CRSRound() {
		return coreCommon.Hash{}
	}
	var s *vm.GovernanceState
	if round == d.CRSRound() {
		s = d.GetHeadState()
	} else {
		s = d.GetStateAtRound(round)
	}
	return coreCommon.Hash(s.CRS())
}

func (d *DexconGovernance) Round() uint64 {
	return d.b.CurrentBlock().Round()
}

func (d *DexconGovernance) CRSRound() uint64 {
	return d.GetHeadState().CRSRound().Uint64()
}

// ProposeCRS send proposals of a new CRS
func (d *DexconGovernance) ProposeCRS(round uint64, signedCRS []byte) {
	data, err := vm.PackProposeCRS(round, signedCRS)
	if err != nil {
		log.Error("failed to pack proposeCRS input", "err", err)
		return
	}

	err = d.sendGovTx(context.Background(), data)
	if err != nil {
		log.Error("failed to send proposeCRS tx", "err", err)
	}
}

// NodeSet returns the current node set.
func (d *DexconGovernance) NodeSet(round uint64) []coreCrypto.PublicKey {
	s := d.GetStateForConfigAtRound(round)
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

// AddDKGComplaint adds a DKGComplaint.
func (d *DexconGovernance) AddDKGComplaint(round uint64, complaint *dkgTypes.Complaint) {
	data, err := vm.PackAddDKGComplaint(round, complaint)
	if err != nil {
		log.Error("failed to pack addDKGComplaint input", "err", err)
		return
	}

	err = d.sendGovTx(context.Background(), data)
	if err != nil {
		log.Error("failed to send addDKGComplaint tx", "err", err)
	}
}

// AddDKGMasterPublicKey adds a DKGMasterPublicKey.
func (d *DexconGovernance) AddDKGMasterPublicKey(round uint64, masterPublicKey *dkgTypes.MasterPublicKey) {
	data, err := vm.PackAddDKGMasterPublicKey(round, masterPublicKey)
	if err != nil {
		log.Error("failed to pack addDKGMasterPublicKey input", "err", err)
		return
	}

	err = d.sendGovTx(context.Background(), data)
	if err != nil {
		log.Error("failed to send addDKGMasterPublicKey tx", "err", err)
	}
}

// AddDKGMPKReady adds a DKG mpk ready message.
func (d *DexconGovernance) AddDKGMPKReady(round uint64, ready *dkgTypes.MPKReady) {
	data, err := vm.PackAddDKGMPKReady(round, ready)
	if err != nil {
		log.Error("failed to pack addDKGMPKReady input", "err", err)
		return
	}

	err = d.sendGovTx(context.Background(), data)
	if err != nil {
		log.Error("failed to send addDKGMPKReady tx", "err", err)
	}
}

// AddDKGFinalize adds a DKG finalize message.
func (d *DexconGovernance) AddDKGFinalize(round uint64, final *dkgTypes.Finalize) {
	data, err := vm.PackAddDKGFinalize(round, final)
	if err != nil {
		log.Error("failed to pack addDKGFinalize input", "err", err)
		return
	}

	err = d.sendGovTx(context.Background(), data)
	if err != nil {
		log.Error("failed to send addDKGFinalize tx", "err", err)
	}
}

// ReportForkVote reports a node for forking votes.
func (d *DexconGovernance) ReportForkVote(vote1, vote2 *coreTypes.Vote) {
	data, err := vm.PackReportForkVote(vote1, vote2)
	if err != nil {
		log.Error("failed to pack report fork vote input", "err", err)
		return
	}

	err = d.sendGovTx(context.Background(), data)
	if err != nil {
		log.Error("failed to send report fork vote tx", "err", err)
	}
}

// ReportForkBlock reports a node for forking blocks.
func (d *DexconGovernance) ReportForkBlock(block1, block2 *coreTypes.Block) {
	data, err := vm.PackReportForkBlock(block1, block2)
	if err != nil {
		log.Error("failed to pack report fork block input", "err", err)
		return
	}

	err = d.sendGovTx(context.Background(), data)
	if err != nil {
		log.Error("failed to send report fork block tx", "err", err)
	}
}

func (d *DexconGovernance) NotarySet(round uint64) (map[string]struct{}, error) {
	notarySet, err := d.nodeSetCache.GetNotarySet(round)
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

func (d *DexconGovernance) ResetDKG(newSignedCRS []byte) {
	data, err := vm.PackResetDKG(newSignedCRS)
	if err != nil {
		log.Error("failed to pack resetDKG input", "err", err)
		return
	}

	err = d.sendGovTx(context.Background(), data)
	if err != nil {
		log.Error("failed to send resetDKG tx", "err", err)
	}
}
