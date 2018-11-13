package reward

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"math/rand"
	"sync"
	"testing"

	"github.com/matrix/go-matrix/reward/util"

	"github.com/matrix/go-matrix/reward/cfg"

	"github.com/matrix/go-matrix/reward/blkreward"

	"github.com/matrix/go-matrix/reward"

	"github.com/matrix/go-matrix/reward/slash"

	"github.com/matrix/go-matrix/crypto"

	"bou.ke/monkey"
	"github.com/matrix/go-matrix/ca"
	"github.com/matrix/go-matrix/common"
	"github.com/matrix/go-matrix/consensus/ethash"
	"github.com/matrix/go-matrix/core"
	"github.com/matrix/go-matrix/core/types"
	"github.com/matrix/go-matrix/core/vm"
	"github.com/matrix/go-matrix/log"
	"github.com/matrix/go-matrix/mc"
	"github.com/matrix/go-matrix/reward/lottery"
	. "github.com/smartystreets/goconvey/convey"
)

const (
	testAddress = "0x8605cdbbdb6d264aa742e77020dcbc58fcdce182"
)

var myNodeId string = "4b2f638f46c7ae5b1564ca7015d716621848a0d9be66f1d1e91d566d2a70eedc2f11e92b743acb8d97dec3fb412c1b2f66afd7fbb9399d4fb2423619eaa51411"

type FakeEth struct {
	blockchain *core.BlockChain
	once       *sync.Once
}

func (s *FakeEth) BlockChain() *core.BlockChain { return s.blockchain }

func fakeEthNew(n int) *FakeEth {
	eth := &FakeEth{once: new(sync.Once)}
	eth.once.Do(func() {
		_, blockchain, err := core.NewCanonical(ethash.NewFaker(), n, true)
		if err != nil {
			fmt.Println("failed to create pristine chain: ", err)
			return
		}
		defer blockchain.Stop()
		eth.blockchain = blockchain
		monkey.Patch(ca.GetTopologyByNumber, func(reqTypes common.RoleType, number uint64) (*mc.TopologyGraph, error) {
			fmt.Println("use monkey  ca.GetTopologyByNumber")
			newGraph := &mc.TopologyGraph{
				Number:        number,
				NodeList:      make([]mc.TopologyNodeInfo, 0),
				CurNodeNumber: 0,
			}
			if common.RoleValidator == reqTypes&common.RoleValidator {
				newGraph.NodeList = append(newGraph.NodeList, mc.TopologyNodeInfo{Account: common.HexToAddress("0x475baee143cf541ff3ee7b00c1c933129238d793"), Position: 8192})
				newGraph.NodeList = append(newGraph.NodeList, mc.TopologyNodeInfo{Account: common.HexToAddress("0x82799145a60b4d1e88d5a895601508f2b7f4ee9b"), Position: 8193})
				newGraph.NodeList = append(newGraph.NodeList, mc.TopologyNodeInfo{Account: common.HexToAddress("0x519437b21e2a0b62788ab9235d0728dd7f1a7269"), Position: 8194})
				newGraph.NodeList = append(newGraph.NodeList, mc.TopologyNodeInfo{Account: common.HexToAddress("0x29216818d3788c2505a593cbbb248907d47d9bce"), Position: 8195})
				newGraph.CurNodeNumber = 4
			}

			return newGraph, nil
		})
		monkey.Patch(ca.GetElectedByHeightAndRole, func(height *big.Int, roleType common.RoleType) ([]vm.DepositDetail, error) {
			fmt.Println("use monkey  ca.GetTopologyByNumber")
			Deposit := make([]vm.DepositDetail, 0)
			if common.RoleValidator == roleType&common.RoleValidator {
				Deposit = append(Deposit, vm.DepositDetail{Address: common.HexToAddress("0x475baee143cf541ff3ee7b00c1c933129238d793"), Deposit: big.NewInt(1e+18)})
				Deposit = append(Deposit, vm.DepositDetail{Address: common.HexToAddress("0x82799145a60b4d1e88d5a895601508f2b7f4ee9b"), Deposit: big.NewInt(2e+18)})
				Deposit = append(Deposit, vm.DepositDetail{Address: common.HexToAddress("0x519437b21e2a0b62788ab9235d0728dd7f1a7269"), Deposit: big.NewInt(3e+18)})
				Deposit = append(Deposit, vm.DepositDetail{Address: common.HexToAddress("0x29216818d3788c2505a593cbbb248907d47d9bce"), Deposit: big.NewInt(4e+18)})
				Deposit = append(Deposit, vm.DepositDetail{Address: common.HexToAddress("0x29216818d3788c2505a593cbbb248907d47d9bcf"), Deposit: big.NewInt(2e+18)})

			}

			return Deposit, nil
		})
		//id, _ := discover.HexID(myNodeId)
		//ca.Start(id, "")
	})
	return eth
}

type InnerSeed struct {
}

func (s *InnerSeed) GetSeed(num uint64) *big.Int {
	random := rand.New(rand.NewSource(0))
	return new(big.Int).SetUint64(random.Uint64())
}
func TestNew(t *testing.T) {
	type args struct {
		chain reward.ChainReader
	}
	log.InitLog(3)
	eth := fakeEthNew(0)
	blkreward.New(eth.blockchain)

}

func TestBlockReward_setLeaderRewards(t *testing.T) {

	log.InitLog(3)
	eth := fakeEthNew(0)
	rewardCfg := cfg.New(nil, nil)
	rewardobject := reward.New(eth.blockchain, rewardCfg)
	Convey("Leader测试", t, func() {
		rewards := make(map[common.Address]*big.Int, 0)
		validatorsBlkReward := util.CalcRateReward(ByzantiumBlockReward, rewardobject.rewardCfg.RewardMount.ValidatorsRate)
		leaderBlkReward := util.CalcRateReward(validatorsBlkReward, rewardobject.rewardCfg.RewardMount.LeaderRate)
		rewardobject.rewardCfg.SetReward.SetLeaderRewards(leaderBlkReward, rewards, common.HexToAddress(testAddress), new(big.Int).SetUint64(uint64(1)))

	})
}

func TestBlockReward_setSelectedBlockRewards(t *testing.T) {
	type args struct {
		chain ChainReader
	}
	log.InitLog(3)
	eth := fakeEthNew(0)
	slash := slash.New(eth.blockchain)
	seed := &InnerSeed{}
	lottery := lottery.New(eth.blockchain, seed)
	reward := New(eth.blockchain, lottery, slash)
	SkipConvey("选中无节点变化测试", t, func() {

		rewards := make(map[common.Address]*big.Int, 0)
		header := eth.BlockChain().CurrentHeader()
		newheader := types.CopyHeader(header)
		newheader.Number = big.NewInt(1)
		newheader.NetTopology.Type = common.NetTopoTypeAll
		newheader.NetTopology.NetTopologyData = append(newheader.NetTopology.NetTopologyData, common.NetTopologyData{Account: common.HexToAddress("0x475baee143cf541ff3ee7b00c1c933129238d793"), Position: 8192})
		newheader.NetTopology.NetTopologyData = append(newheader.NetTopology.NetTopologyData, common.NetTopologyData{Account: common.HexToAddress("0x82799145a60b4d1e88d5a895601508f2b7f4ee9b"), Position: 8193})
		newheader.NetTopology.NetTopologyData = append(newheader.NetTopology.NetTopologyData, common.NetTopologyData{Account: common.HexToAddress("0x519437b21e2a0b62788ab9235d0728dd7f1a7269"), Position: 8194})
		newheader.NetTopology.NetTopologyData = append(newheader.NetTopology.NetTopologyData, common.NetTopologyData{Account: common.HexToAddress("0x29216818d3788c2505a593cbbb248907d47d9bce"), Position: 8195})
		reward.setSelectedBlockRewards(reward.electedValidatorsReward, rewards, common.RoleValidator|common.RoleBackupValidator, newheader, BackupRewardRate)
		//So(rewards[common.HexToAddress(testAddress)], ShouldEqual, reward.leaderBlkReward)
	})

	Convey("选中有节点变化测试", t, func() {

		rewards := make(map[common.Address]*big.Int, 0)
		header := eth.BlockChain().CurrentHeader()
		newheader := types.CopyHeader(header)
		newheader.Number = big.NewInt(1)
		newheader.NetTopology.Type = common.NetTopoTypeAll
		newheader.NetTopology.NetTopologyData = append(newheader.NetTopology.NetTopologyData, common.NetTopologyData{Account: common.HexToAddress("0x475baee143cf541ff3ee7b00c1c933129238d793"), Position: 8192})
		newheader.NetTopology.NetTopologyData = append(newheader.NetTopology.NetTopologyData, common.NetTopologyData{Account: common.HexToAddress("0x82799145a60b4d1e88d5a895601508f2b7f4ee9b"), Position: 8193})
		newheader.NetTopology.NetTopologyData = append(newheader.NetTopology.NetTopologyData, common.NetTopologyData{Account: common.HexToAddress("0x519437b21e2a0b62788ab9235d0728dd7f1a7269"), Position: 8194})
		newheader.NetTopology.NetTopologyData = append(newheader.NetTopology.NetTopologyData, common.NetTopologyData{Account: common.HexToAddress("0x29216818d3788c2505a593cbbb248907d47d9bcf"), Position: 8195})
		reward.setSelectedBlockRewards(reward.electedValidatorsReward, rewards, common.RoleValidator|common.RoleBackupValidator, newheader, BackupRewardRate)
		//So(rewards[common.HexToAddress(testAddress)], ShouldEqual, reward.leaderBlkReward)
	})
}

func TestBlockReward_calcTxsFees(t *testing.T) {
	Convey("计算交易费", t, func() {

		log.InitLog(3)
		eth := fakeEthNew(0)
		slash := slash.New(eth.blockchain)
		seed := &InnerSeed{}
		lottery := lottery.New(eth.blockchain, seed)
		reward := New(eth.blockchain, lottery, slash)
		keys := make([]*ecdsa.PrivateKey, 25)
		for i := 0; i < len(keys); i++ {
			keys[i], _ = crypto.GenerateKey()
		}

		signer := types.HomesteadSigner{}
		// Generate a batch of transactions with overlapping values, but shifted nonces
		groups := map[common.Address]types.Transactions{}
		for start, key := range keys {
			addr := crypto.PubkeyToAddress(key.PublicKey)
			for i := 0; i < 25; i++ {
				tx, _ := types.SignTx(types.NewTransaction(uint64(start+i), common.Address{}, big.NewInt(100), 100, big.NewInt(int64(100)), nil), signer, key)
				groups[addr] = append(groups[addr], tx)
			}
		}
		txset := types.NewTransactionsByPriceAndNonce(signer, groups)

		txs := types.Transactions{}
		for tx := txset.Peek(); tx != nil; tx = txset.Peek() {
			txs = append(txs, tx)
			txset.Shift()
		}
		rewards := util.CalcTxsFees(eth.blockchain, big.NewInt(int64(0)), txs)

		So(rewards.Uint64(), ShouldEqual, 1312500000)
	})
}
