// Copyright (c) 2018 The MATRIX Authors
// Distributed under the MIT software license, see the accompanying
// file COPYING or or http://www.opensource.org/licenses/mit-license.php

package core

import (
	"fmt"

	"github.com/matrix/go-matrix/common"
	"github.com/matrix/go-matrix/consensus"
	"github.com/matrix/go-matrix/core/state"
	"github.com/matrix/go-matrix/core/types"
	"github.com/matrix/go-matrix/params"
)

// BlockValidator is responsible for validating block headers, uncles and
// processed state.
//
// BlockValidator implements Validator.
type BlockValidator struct {
	config *params.ChainConfig // Chain configuration options
	bc     *BlockChain         // Canonical block chain
	engine consensus.Engine    // Consensus engine used for validating
}

// NewBlockValidator returns a new block validator which is safe for re-use
func NewBlockValidator(config *params.ChainConfig, blockchain *BlockChain, engine consensus.Engine) *BlockValidator {
	validator := &BlockValidator{
		config: config,
		engine: engine,
		bc:     blockchain,
	}
	return validator
}

// ValidateBody validates the given block's uncles and verifies the the block
// header's transaction and uncle roots. The headers are assumed to be already
// validated at this point.
func (v *BlockValidator) ValidateBody(block *types.Block) error {
	// Check whether the block's known, and if not, that it's linkable
	if v.bc.HasBlockAndState(block.Hash(), block.NumberU64()) {
		return ErrKnownBlock
	}
	if !v.bc.HasBlockAndState(block.ParentHash(), block.NumberU64()-1) {
		if !v.bc.HasBlock(block.ParentHash(), block.NumberU64()-1) {
			return consensus.ErrUnknownAncestor
		}
		return consensus.ErrPrunedAncestor
	}
	// Header validity is known at this point, check the uncles and transactions
	header := block.Header()
	if err := v.engine.VerifyUncles(v.bc, block); err != nil {
		return err
	}
	if hash := types.CalcUncleHash(block.Uncles()); hash != header.UncleHash {
		return fmt.Errorf("uncle root hash mismatch: have %x, want %x", hash, header.UncleHash)
	}
	for _, currencie := range block.Currencies() {
		for _, head := range header.Roots {
			if head.Cointyp == currencie.CurrencyName {
				if hash := types.DeriveSha(types.SelfTransactions(currencie.Transactions.GetTransactions())); hash != head.TxHash {
					return fmt.Errorf("transaction root hash mismatch: have %x, want %x", hash, head.TxHash)
				}
			}
		}
	}
	return nil
}

// ValidateState validates the various changes that happen after a state
// transition, such as amount of used gas, the receipt roots and the state root
// itself. ValidateState returns a database batch if the validation was a success
// otherwise nil and an error is returned.
func (v *BlockValidator) ValidateState(block, parent *types.Block, statedb *state.StateDBManage, usedGas uint64) error {
	header := block.Header()
	if block.GasUsed() != usedGas {
		return fmt.Errorf("invalid gas used (remote: %d local: %d)", block.GasUsed(), usedGas)
	}
	// Validate the received block's bloom with the one derived from the generated receipts.
	// For valid blocks this should always validate to true.
	for _, currencie := range block.Currencies() {
		ct,rbloom := currencie.CurrencyName,types.CreateBloom(currencie.Receipts.GetReceipts())
		receiptSha := types.DeriveSha(currencie.Receipts.GetReceipts())
		for _,cr:=range header.Roots{
			if cr.Cointyp==ct {
				if rbloom != cr.Bloom {
					return fmt.Errorf("invalid bloom (remote: %x  local: %x)", cr.Bloom, rbloom)
				}
		// Tre receipt Trie's root (R = (Tr [[H1, R1], ... [Hn, R1]]))
				if receiptSha != cr.ReceiptHash {
					return fmt.Errorf("invalid receipt root hash (remote: %x local: %x)", cr.ReceiptHash, receiptSha)
					}
				break
			}
		}
	}
	// Validate the state root against the received state root and throw
	// an error if they don't match.
	var root []common.CoinRoot
	//var Coinbyte	[] common.Coinbyte
	root, _ = statedb.IntermediateRoot(v.config.IsEIP158(header.Number))
	if types.RlpHash(root) != types.RlpHash(header.Roots) { //ShardingYY
		return fmt.Errorf("invalid merkle root (remote: %x local: %x)", header.Roots, root)
	}
	//if types.RlpHash(Coinbyte)	!=types.RlpHash(header.Sharding){//ShardingBB
	//	return fmt.Errorf("invalid merkle Coinbyte (remote: %x local: %x)", header.Sharding, Coinbyte)
	//}

	return nil
}

// CalcGasLimit computes the gas limit of the next block after parent.
// This is miner strategy, not consensus protocol.
func CalcGasLimit(parent *types.Block) uint64 {
	// contrib = (parentGasUsed * 3 / 2) / 1024
	contrib := (parent.GasUsed() + parent.GasUsed()/2) / params.GasLimitBoundDivisor

	// decay = parentGasLimit / 1024 -1
	decay := parent.GasLimit()/params.GasLimitBoundDivisor - 1

	/*
		strategy: gasLimit of block-to-mine is set based on parent's
		gasUsed value.  if parentGasUsed > parentGasLimit * (2/3) then we
		increase it, otherwise lower it (or leave it unchanged if it's right
		at that usage) the amount increased/decreased depends on how far away
		from parentGasLimit * (2/3) parentGasUsed is.
	*/
	limit := parent.GasLimit() - decay + contrib
	if limit < params.MinGasLimit {
		limit = params.MinGasLimit
	}
	// however, if we're now below the target (TargetGasLimit) we increase the
	// limit as much as we can (parentGasLimit / 1024 -1)
	if limit < params.TargetGasLimit {
		limit = parent.GasLimit() + decay
		if limit > params.TargetGasLimit {
			limit = params.TargetGasLimit
		}
	}
	return limit
}
