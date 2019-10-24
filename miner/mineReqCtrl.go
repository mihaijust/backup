// Copyright (c) 2018 The MATRIX Authors
// Distributed under the MIT software license, see the accompanying
// file COPYING or or http://www.opensource.org/licenses/mit-license.php

package miner

import (
	"math/big"
	"sync"

	"github.com/MatrixAINetwork/go-matrix/mc"

	"github.com/MatrixAINetwork/go-matrix/ca"
	"github.com/MatrixAINetwork/go-matrix/common"
	"github.com/MatrixAINetwork/go-matrix/consensus"
	"github.com/MatrixAINetwork/go-matrix/core/types"
	"github.com/MatrixAINetwork/go-matrix/log"
	"github.com/pkg/errors"
	"github.com/MatrixAINetwork/go-matrix/params/manparams"
)

type mineReqData struct {
	mu                 sync.Mutex
	coinbase		   common.Address
	mined              bool
	headerHash         common.Hash
	header             *types.Header
	isBroadcastReq     bool
	isFriend		   bool
	txs                []types.CoinSelfTransaction
	mineDiff           *big.Int
	mineResultSendTime int64
}

func newMineReqData(headerHash common.Hash, header *types.Header, txs []types.CoinSelfTransaction, isBroadcastReq bool,isfriend bool) *mineReqData {
	return &mineReqData{
		mined:              false,
		headerHash:         headerHash,
		header:             header,
		isBroadcastReq:     isBroadcastReq,
		isFriend:           isfriend,
		txs:                txs,
		mineDiff:           nil,
		mineResultSendTime: 0,
	}
}

func (self *mineReqData) ResendMineResult(curTime int64) error {
	self.mu.Lock()
	defer self.mu.Unlock()
	if curTime-self.mineResultSendTime < manparams.MinerResultSendInterval {
		return errors.Errorf("挖矿发送间隔尚未到, 上次发送时间(%d), 当前时间(%d)", self.mineResultSendTime, curTime)
	}
	self.mineResultSendTime = curTime
	return nil
}

type mineReqCtrl struct {
	curSuperSeq     uint64
	curNumber       uint64
	currentMineReq  *mineReqData
	role            common.RoleType
	bcInterval      *mc.BCIntervalInfo
	bc              ChainReader
	validatorReader consensus.StateReader
	reqCache        map[common.Hash]*mineReqData
	futureReq       map[uint64][]*mineReqData //todo 考虑作恶，可以加入限长
}

func newMinReqCtrl(bc ChainReader) *mineReqCtrl {
	return &mineReqCtrl{
		curSuperSeq:     0,
		curNumber:       0,
		currentMineReq:  nil,
		role:            common.RoleNil,
		bcInterval:      nil,
		validatorReader: bc,
		bc:              bc,
		reqCache:        make(map[common.Hash]*mineReqData),
		futureReq:       make(map[uint64][]*mineReqData),
	}
}

func (ctrl *mineReqCtrl) Clear() {
	ctrl.curNumber = 0
	ctrl.role = common.RoleNil
	ctrl.bcInterval = nil
	ctrl.currentMineReq = nil
	ctrl.reqCache = make(map[common.Hash]*mineReqData)
	ctrl.futureReq = make(map[uint64][]*mineReqData)
	return
}

func (ctrl *mineReqCtrl) SetNewNumber(number uint64, role common.RoleType) {
	if ctrl.curNumber > number {
		return
	}

	ctrl.role = common.RoleMiner
	/*
	bcInterval, err := manparams.GetBCIntervalInfoByNumber(number - 1)
	if err != nil {
		log.ERROR("miner ctrl", "获取广播周期失败", err)
	} else {
		ctrl.bcInterval = bcInterval
	}
*/
	if ctrl.curNumber < number {
		ctrl.curNumber = number
		ctrl.reqCache = make(map[common.Hash]*mineReqData)
	}
	return
}

func (ctrl *mineReqCtrl) AddMineReq(header *types.Header, miner common.Address, isBroadcastReq bool,isfriend bool) (*mineReqData, error) {
	if nil == header {
		return nil, errors.New("header为nil")
	}

	reqNumber := header.Number.Uint64()
	headerHash := header.HashNoSignsAndNonce()
	if reqNumber > ctrl.curNumber{
		ctrl.SetNewNumber(reqNumber,common.RoleMiner)
	}
	/*
	if reqNumber > ctrl.curNumber {
		list, exist := ctrl.futureReq[reqNumber]
		reqData := newMineReqData(headerHash, header, txs, isBroadcastReq)
		if exist {
			ctrl.futureReq[reqNumber] = append(list, reqData)
		} else {
			ctrl.futureReq[reqNumber] = []*mineReqData{reqData}
		}
		return reqData, nil
	} else */if reqNumber < ctrl.curNumber {
		return nil, errors.Errorf("挖矿请求消息高度(%d) 小于 当前高度(%d)", reqNumber, ctrl.curNumber)
	} else {

		data, exist := ctrl.reqCache[headerHash]
		if exist {
			return data, nil
		}

		if err := ctrl.checkMineReq(header); err != nil {
			return nil, err
		}

		reqData := newMineReqData(headerHash, header, nil, isBroadcastReq,isfriend)
		reqData.coinbase = miner
		ctrl.reqCache[headerHash] = reqData
		return reqData, nil
	}
}

func (ctrl *mineReqCtrl) CanMining() bool {
	return true
	return ctrl.roleCanMine(ctrl.role, ctrl.curNumber)
}

func (ctrl *mineReqCtrl) GetMineReqData(headerHash common.Hash) (*mineReqData, error) {
	reqData, exist := ctrl.reqCache[headerHash]
	if !exist {
		return nil, errors.New("请求消息未找到")
	}
	if reqData == nil {
		return nil, errors.New("请求消息找到，但是为nil")
	}
	return reqData, nil
}

func (ctrl *mineReqCtrl) GetUnMinedReq() *mineReqData {
	//todo 获取时间戳最大的

	removeHash := make([]common.Hash,0,len(ctrl.reqCache))
	var reqMax *mineReqData
	for hash, req := range ctrl.reqCache {
		reqNumber := req.header.Number.Uint64()
		if reqNumber<ctrl.curNumber{
			if reqNumber<ctrl.curNumber-1 {
			removeHash = append(removeHash,hash)
			}
			continue
		}
		if req == nil {
			log.ERROR(ModuleMiner, "GetUnMinedReq", "有reqData为nil", "hash", hash.TerminalString())
			removeHash = append(removeHash,hash)
			continue
		}
		if req.mined {
//			removeHash = append(removeHash,hash)
			continue
		}
		if reqMax==nil{
			reqMax = req
		}else if reqMax.header.Number.Cmp(req.header.Number)<0{
			reqMax = req
		}
	}
	for _,hash := range removeHash{
		delete (ctrl.reqCache,hash)
	}
	return reqMax
}

func (ctrl *mineReqCtrl) SetCurrentMineReq(headerHash common.Hash) error {
	if ctrl.currentMineReq != nil && ctrl.currentMineReq.headerHash == headerHash {
		return nil
	}
	req, err := ctrl.GetMineReqData(headerHash)
	if err != nil {
		return err
	}
	if req.mined {
		return errors.Errorf("请求(%s)已挖矿完成", headerHash.TerminalString())
	}
	ctrl.currentMineReq = req
	return nil
}

func (ctrl *mineReqCtrl) GetCurrentMineReq() *mineReqData {
	return ctrl.currentMineReq
}

func (ctrl *mineReqCtrl) SetMiningResult(result *types.Header) (*mineReqData, error) {
	if nil == result {
		return nil, errors.New("消息为nil")
	}
	headerHash := result.HashNoSignsAndNonce()
	req, err := ctrl.GetMineReqData(headerHash)
	if err != nil {
		return nil, err
	}

	if req.mined {
		return nil, errors.Errorf("请求(%s)已挖矿完成", headerHash.TerminalString())
	}

	req.mineDiff = result.Difficulty

	if req.isBroadcastReq {
		req.header.Coinbase = ca.GetDepositAddress()
	} else {
		req.header.Nonce = result.Nonce
		req.header.Coinbase = result.Coinbase
		req.header.MixDigest = result.MixDigest
		req.header.Signatures = result.Signatures
	}

	req.mined = true
	if ctrl.currentMineReq != nil && ctrl.currentMineReq.headerHash == headerHash {
		ctrl.currentMineReq = nil
	}
	return req, nil
}

func (ctrl *mineReqCtrl) checkMineReq(header *types.Header) error {
	return nil
	if header.Difficulty.Uint64() == 0 {
		return difficultyIsZero
	}

	err := ctrl.bc.DPOSEngine(header.Version).VerifyBlock(ctrl.validatorReader, header)
	if err != nil {
		return errors.Errorf("挖矿请求POS验证失败(%v)", err)
	}
	return nil
}

func (ctrl *mineReqCtrl) roleCanMine(role common.RoleType, number uint64) bool {
	if ctrl.bcInterval == nil {
		return false
	}

	if ctrl.bcInterval.IsBroadcastNumber(number) {
		return role == common.RoleBroadcast
	} else {
		return role == common.RoleMiner || role == common.RoleInnerMiner
	}
}

func (ctrl *mineReqCtrl) fixMap() {
	ctrl.reqCache = make(map[common.Hash]*mineReqData)
	reqList, exist := ctrl.futureReq[ctrl.curNumber]
	if !exist {
		return
	}

	for i := 0; i < len(reqList); i++ {
		reqData := reqList[i]
		_, exist := ctrl.reqCache[reqData.headerHash]
		if exist {
			continue
		}
		err := ctrl.checkMineReq(reqData.header)
		if err != nil {
			log.WARN(ModuleMiner, "fixMap", "检测请求时，验证失败", err, "高度", ctrl.curNumber)
			continue
		}
		ctrl.reqCache[reqData.headerHash] = reqData
	}

	delete(ctrl.futureReq, ctrl.curNumber)
}
