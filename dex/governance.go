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
	"math/big"

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

	b           *DexAPIBackend
	chainConfig *params.ChainConfig
	privateKey  *ecdsa.PrivateKey
	address     common.Address
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

	gasLimit, err := core.IntrinsicGas(data, false, false)
	if err != nil {
		return err
	}

	tx := types.NewTransaction(
		nonce,
		vm.GovernanceContractAddress,
		big.NewInt(0),
		gasLimit+vm.GovernanceActionGasCost,
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

func (d *DexconGovernance) Round() uint64 {
	return d.b.CurrentBlock().Round()
}

// ProposeCRS send proposals of a new CRS
func (d *DexconGovernance) ProposeCRS(round uint64, signedCRS []byte) {
	data, err := vm.PackProposeCRS(round, signedCRS)
	if err != nil {
		log.Error("Failed to pack proposeCRS input", "err", err)
		return
	}

	err = d.sendGovTx(context.Background(), data)
	if err != nil {
		log.Error("Failed to send proposeCRS tx", "err", err)
	}
}

// AddDKGComplaint adds a DKGComplaint.
func (d *DexconGovernance) AddDKGComplaint(complaint *dkgTypes.Complaint) {
	data, err := vm.PackAddDKGComplaint(complaint)
	if err != nil {
		log.Error("Failed to pack addDKGComplaint input", "err", err)
		return
	}

	err = d.sendGovTx(context.Background(), data)
	if err != nil {
		log.Error("Failed to send addDKGComplaint tx", "err", err)
	}
}

// AddDKGMasterPublicKey adds a DKGMasterPublicKey.
func (d *DexconGovernance) AddDKGMasterPublicKey(masterPublicKey *dkgTypes.MasterPublicKey) {
	data, err := vm.PackAddDKGMasterPublicKey(masterPublicKey)
	if err != nil {
		log.Error("Failed to pack addDKGMasterPublicKey input", "err", err)
		return
	}

	err = d.sendGovTx(context.Background(), data)
	if err != nil {
		log.Error("Failed to send addDKGMasterPublicKey tx", "err", err)
	}
}

// AddDKGMPKReady adds a DKG mpk ready message.
func (d *DexconGovernance) AddDKGMPKReady(ready *dkgTypes.MPKReady) {
	data, err := vm.PackAddDKGMPKReady(ready)
	if err != nil {
		log.Error("Failed to pack addDKGMPKReady input", "err", err)
		return
	}

	err = d.sendGovTx(context.Background(), data)
	if err != nil {
		log.Error("Failed to send addDKGMPKReady tx", "err", err)
	}
}

// AddDKGFinalize adds a DKG finalize message.
func (d *DexconGovernance) AddDKGFinalize(final *dkgTypes.Finalize) {
	data, err := vm.PackAddDKGFinalize(final)
	if err != nil {
		log.Error("Failed to pack addDKGFinalize input", "err", err)
		return
	}

	err = d.sendGovTx(context.Background(), data)
	if err != nil {
		log.Error("Failed to send addDKGFinalize tx", "err", err)
	}
}

// AddDKGSuccess adds a DKG success message.
func (d *DexconGovernance) AddDKGSuccess(success *dkgTypes.Success) {
	data, err := vm.PackAddDKGSuccess(success)
	if err != nil {
		log.Error("Failed to pack addDKGSuccess input", "err", err)
		return
	}

	err = d.sendGovTx(context.Background(), data)
	if err != nil {
		log.Error("Failed to send addDKGSuccess tx", "err", err)
	}
}

// ReportForkVote reports a node for forking votes.
func (d *DexconGovernance) ReportForkVote(vote1, vote2 *coreTypes.Vote) {
	data, err := vm.PackReportForkVote(vote1, vote2)
	if err != nil {
		log.Error("Failed to pack report fork vote input", "err", err)
		return
	}

	err = d.sendGovTx(context.Background(), data)
	if err != nil {
		log.Error("Failed to send report fork vote tx", "err", err)
	}
}

// ReportForkBlock reports a node for forking blocks.
func (d *DexconGovernance) ReportForkBlock(block1, block2 *coreTypes.Block) {
	data, err := vm.PackReportForkBlock(block1, block2)
	if err != nil {
		log.Error("Failed to pack report fork block input", "err", err)
		return
	}

	err = d.sendGovTx(context.Background(), data)
	if err != nil {
		log.Error("Failed to send report fork block tx", "err", err)
	}
}

func (d *DexconGovernance) ResetDKG(newSignedCRS []byte) {
	data, err := vm.PackResetDKG(newSignedCRS)
	if err != nil {
		log.Error("Failed to pack resetDKG input", "err", err)
		return
	}

	err = d.sendGovTx(context.Background(), data)
	if err != nil {
		log.Error("Failed to send resetDKG tx", "err", err)
	}
}
