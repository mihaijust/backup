// Copyright (c) 2018 The MATRIX Authors
// Distributed under the MIT software license, see the accompanying
// file COPYING or or http://www.opensource.org/licenses/mit-license.php
package leaderelect

import (
	"time"

	"github.com/matrix/go-matrix/ca"
	"github.com/matrix/go-matrix/common"
	"github.com/matrix/go-matrix/log"
	"github.com/matrix/go-matrix/mc"
)

func (self *controller) handleMsg(data interface{}) {
	if nil == data {
		log.WARN(self.logInfo, "消息处理", "收到nil消息")
		return
	}

	switch data.(type) {
	case *startControllerMsg:
		msg, _ := data.(*startControllerMsg)
		self.handleStartMsg(msg)

	case *mc.BlockPOSFinishedNotify:
		msg, _ := data.(*mc.BlockPOSFinishedNotify)
		self.handleBlockPOSFinishedNotify(msg)

	case *mc.HD_ReelectInquiryReqMsg:
		msg, _ := data.(*mc.HD_ReelectInquiryReqMsg)
		self.handleInquiryReq(msg)

	case *mc.HD_ReelectInquiryRspMsg:
		msg, _ := data.(*mc.HD_ReelectInquiryRspMsg)
		self.handleInquiryRsp(msg)

	case *mc.HD_ReelectLeaderReqMsg:
		msg, _ := data.(*mc.HD_ReelectLeaderReqMsg)
		self.handleRLReq(msg)

	case *mc.HD_ConsensusVote:
		msg, _ := data.(*mc.HD_ConsensusVote)
		self.handleRLVote(msg)

	case *mc.HD_ReelectBroadcastMsg:
		msg, _ := data.(*mc.HD_ReelectBroadcastMsg)
		self.handleBroadcastMsg(msg)

	case *mc.HD_ReelectBroadcastRspMsg:
		msg, _ := data.(*mc.HD_ReelectBroadcastRspMsg)
		self.handleBroadcastRsp(msg)

	default:
		log.WARN(self.logInfo, "消息处理", "未知消息类型")
	}
}

func (self *controller) SetSelfAddress(addr common.Address, nodeAddr common.Address) {
	self.dc.selfAddr = addr
	self.dc.selfNodeAddr = nodeAddr
	self.selfCache.selfAddr = addr
	self.selfCache.selfNodeAddr = nodeAddr
}

func (self *controller) handleStartMsg(msg *startControllerMsg) {
	if nil == msg || nil == msg.parentHeader {
		log.WARN(self.logInfo, "开始消息处理", ErrParamsIsNil)
		return
	}

	a0Address := ca.GetDepositAddress()
	nodeAddress := ca.GetSignAddress()
	self.SetSelfAddress(a0Address, nodeAddress)
	log.Info("测试测试测试", "selfDepositAddress", a0Address.String(), "nodeAddress", nodeAddress.String())

	log.INFO(self.logInfo, "开始消息处理", "start", "高度", self.dc.number, "isSupper", msg.parentIsSupper, "preLeader", msg.parentHeader.Leader.Hex(), "header time", msg.parentHeader.Time.Int64())
	if err := self.dc.AnalysisState(msg.parentHeader, msg.parentIsSupper, msg.parentStateDB); err != nil {
		log.ERROR(self.logInfo, "开始消息处理", "分析状态树信息错误", "err", err)
		return
	}

	if self.dc.role != common.RoleValidator {
		log.Debug(self.logInfo, "开始消息处理", "身份错误, 不是验证者", "高度", self.dc.number)
		self.mp.SaveParentHeader(msg.parentHeader)
		return
	}

	if self.dc.bcInterval.IsBroadcastNumber(self.dc.number) {
		log.Debug(self.logInfo, "开始消息处理", "区块为广播区块，不开启定时器")
		self.dc.state = stIdle
		self.publishLeaderMsg()
		self.mp.SaveParentHeader(msg.parentHeader)
		return
	}

	if self.dc.turnTime.SetBeginTime(mc.ConsensusTurnInfo{}, msg.parentHeader.Time.Int64()) {
		log.Debug(self.logInfo, "开始消息处理", "更新轮次时间成功", "高度", self.dc.number)
		self.dc.leaderCal.dumpAllValidators(self.logInfo)
		self.mp.SaveParentHeader(msg.parentHeader)
		if isFirstConsensusTurn(self.ConsensusTurn()) {
			curTime := time.Now().Unix()
			st, remainTime, reelectTurn := self.dc.turnTime.CalState(mc.ConsensusTurnInfo{}, curTime)
			log.INFO(self.logInfo, "开始消息处理", "完成", "状态计算结果", st.String(), "剩余时间", remainTime, "重选轮次", reelectTurn)
			self.dc.state = st
			self.dc.curReelectTurn = 0
			self.setTimer(remainTime, self.timer)
			if st == stPos {
				self.processPOSState()
			} else if st == stReelect {
				self.startReelect(reelectTurn)
			}
		}
	}

	//公布leader身份
	self.publishLeaderMsg()
}

func (self *controller) handleBlockPOSFinishedNotify(msg *mc.BlockPOSFinishedNotify) {
	if nil == msg || nil == msg.Header {
		log.WARN(self.logInfo, "POS完成通知消息处理", ErrParamsIsNil)
		return
	}
	if err := self.mp.SavePOSNotifyMsg(msg); err == nil {
		log.Info(self.logInfo, "POS完成通知消息处理", "缓存成功", "高度", msg.Number, "leader", msg.Header.Leader, "leader轮次", msg.ConsensusTurn.String())
	}
	self.processPOSState()
}

func (self *controller) timeOutHandle() {
	curTime := time.Now().Unix()
	st, remainTime, reelectTurn := self.dc.turnTime.CalState(self.dc.curConsensusTurn, curTime)
	switch self.State() {
	case stPos:
		log.Warn(self.logInfo, "超时事件", "POS未完成", "轮次", self.curTurnInfo(), "高度", self.Number(),
			"状态计算结果", st.String(), "下次超时时间", remainTime, "计算的重选轮次", reelectTurn,
			"轮次开始时间", self.dc.turnTime.GetBeginTime(*self.ConsensusTurn()), "leader", self.dc.GetConsensusLeader().Hex())
	case stReelect:
		log.Warn(self.logInfo, "超时事件", "重选未完成", "轮次", self.curTurnInfo(), "高度", self.Number(),
			"状态计算结果", st.String(), "下次超时时间", remainTime, "计算的重选轮次", reelectTurn,
			"轮次开始时间", self.dc.turnTime.GetBeginTime(*self.ConsensusTurn()), "master", self.dc.GetReelectMaster().Hex())
	default:
		log.ERROR(self.logInfo, "超时事件", "当前状态错误", "state", self.State().String(), "轮次", self.curTurnInfo(), "高度", self.Number(),
			"轮次开始时间", self.dc.turnTime.GetBeginTime(*self.ConsensusTurn()), "当前时间", curTime)
		return
	}

	self.setTimer(remainTime, self.timer)
	self.dc.state = st
	self.startReelect(reelectTurn)
}

func (self *controller) processPOSState() {
	if self.State() != stPos {
		log.Debug(self.logInfo, "执行检查POS状态", "状态不正常,不执行", "当前状态", self.State().String())
		return
	}

	if _, err := self.mp.GetPOSNotifyMsg(self.dc.GetConsensusLeader(), self.dc.curConsensusTurn); err != nil {
		log.Debug(self.logInfo, "执行检查POS状态", "获取POS完成消息失败", "err", err)
		return
	}

	log.Info(self.logInfo, "POS完成", "状态切换为<挖矿结果等待阶段>")
	self.setTimer(0, self.timer)
	self.dc.state = stMining
}
