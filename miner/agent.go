// Copyright (c) 2008 The MATRIX Authors 
// Distributed under the MIT software license, see the accompanying
// file COPYING or or http://www.opensource.org/licenses/mit-license.php
// Copyright 2015 The go-matrix Authors
// This file is part of the go-matrix library.
//
// The go-matrix library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-matrix library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-matrix library. If not, see <http://www.gnu.org/licenses/>.

package miner

import (
	"sync"

	"sync/atomic"

	"github.com/matrix/go-matrix/consensus"
	"github.com/matrix/go-matrix/log"
)

type CpuAgent struct {
	mu sync.Mutex

	workCh        chan *Work
	stop          chan struct{}
	quitCurrentOp chan struct{}
	stopMineCh    chan struct{}
	returnCh      chan<- *Result
	foundMsgCh    chan *consensus.FoundMsg

	chain  consensus.ChainReader
	engine consensus.Engine

	isMining int32 // isMining indicates whether the agent is currently mining
}

func NewCpuAgent(chain consensus.ChainReader, engine consensus.Engine) *CpuAgent {
	miner := &CpuAgent{
		chain:      chain,
		engine:     engine,
		stop:       make(chan struct{}, 1),
		workCh:     make(chan *Work, 1),
		foundMsgCh: make(chan *consensus.FoundMsg, 1),
		stopMineCh: make(chan struct{}, 1),
	}
	return miner
}

func (self *CpuAgent) Work() chan<- *Work            { return self.workCh }
func (self *CpuAgent) SetReturnCh(ch chan<- *Result) { self.returnCh = ch }

func (self *CpuAgent) Stop() {
	if !atomic.CompareAndSwapInt32(&self.isMining, 1, 0) {
		return // agent already stopped
	}
	self.stop <- struct{}{}
done:
	// Empty work channel
	for {
		select {
		case <-self.workCh:
		default:
			break done
		}
	}
}

func (self *CpuAgent) Start() {
	if !atomic.CompareAndSwapInt32(&self.isMining, 0, 1) {
		return // agent already started
	}
	go self.update()
}

func (self *CpuAgent) update() {
out:
	for {
		select {
		case work := <-self.workCh:
			self.mu.Lock()
			if self.quitCurrentOp != nil {
				close(self.quitCurrentOp)
			}
			self.quitCurrentOp = make(chan struct{})
			go self.mine(work, self.quitCurrentOp)
			self.mu.Unlock()
		case <-self.stop:
			self.mu.Lock()
			if self.quitCurrentOp != nil {
				log.Info("miner", "CpuAgent close quitCurrentOp", "")
				close(self.quitCurrentOp)
				self.quitCurrentOp = nil
			}
			self.mu.Unlock()
			log.Info("miner", "CpuAgent Stop Minning", "")

			break out
		}
	}
}

func (self *CpuAgent) mine(work *Work, stop <-chan struct{}) {
	//if result, err := self.engine.Seal(self.chain, work.header, stop, work.difficultyList, work.isBroadcastNode); result != nil {
	//	self.returnCh <- &Result{result.Difficulty, result.Header}
	//} else {
	//	if err != nil {
	//		log.Warn("Block sealing failed", "err", err)
	//	}
	//	self.returnCh <- nil
	//}

	go self.engine.Seal(self.chain, work.header, stop, self.foundMsgCh, work.difficultyList, work.isBroadcastNode)

	for {
		select {
		case result := <-self.foundMsgCh:
			self.returnCh <- &Result{result.Difficulty, result.Header}
		case <-self.stopMineCh:
			log.Info("miner", "quit agent mine")
			return
		}
	}
}

func (self *CpuAgent) GetHashRate() int64 {
	if pow, ok := self.engine.(consensus.PoW); ok {
		return int64(pow.Hashrate())
	}
	return 0
}
