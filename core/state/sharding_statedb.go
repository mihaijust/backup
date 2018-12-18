package state

import (
	"github.com/matrix/go-matrix/common"
	"github.com/matrix/go-matrix/core/types"
	"github.com/matrix/go-matrix/crypto"
	"github.com/matrix/go-matrix/log"
	"github.com/matrix/go-matrix/params"
	"github.com/matrix/go-matrix/rlp"
	"github.com/matrix/go-matrix/trie"
	"math/big"
	"fmt"
	"sort"
	"encoding/json"
)
const (
	sharding_MOUNTS=256
)
type CoinRoot struct {
	Cointyp string
	root    common.Hash
}
type RangeRootManage struct {
	Range   byte
	//	State 	*StateDB
	root common.Hash
}
type RangeManage struct {
	Range   byte
	State 	*StateDB
	//root common.Hash
}
type CoinManage struct {
	Cointyp string
	ValSeg  []*RangeManage
}
type ShardingStateDB struct {
	db          Database
	trie        Trie
	shardings	[]*CoinManage
	coinRoot    []CoinRoot
}

// Create a new state from a given trie.
func NewSharding(roots []CoinRoot, db Database) (*ShardingStateDB, error) {
	b,_ := json.Marshal(roots)
	hash := common.BytesToHash(b)
	tree, err := db.OpenTrie(hash)
	if err != nil {
		return nil, err
	}
	return &ShardingStateDB{
		db:                db,
		trie:              tree,
		shardings:         make([]*CoinManage,0),
		coinRoot:          roots,
	}, nil
}
//func (shard *ShardingStateDB) SetStatedb(typ []string,b []byte){
//
//}
func (shard *ShardingStateDB) MakeStatedb(cointyp string,b byte) {
	//没有对应币种或byte分区的时候，才创建
	for _,sh := range shard.shardings{
		if sh.Cointyp == cointyp{
			for _,st := range sh.ValSeg{
				if st.Range == b{
					return
				}
			}
		}
	}
	var root common.Hash
	//获取指定的币种root
	for _,cr := range shard.coinRoot{
		if cr.Cointyp == cointyp{
			root = cr.root
		}
	}
	var root256 []common.Hash
	err:=json.Unmarshal(root[:],&root256)
	if err != nil{
		log.Error("file sharding_statedb", "func MakeStatedb:Unmarshal:root", err)
		panic(err)
	}
	rms := make([]RangeManage,0)
	for _,rt := range root256{
		//获取rangemanage
		rm_b,err := shard.trie.TryGet(rt[:])
		if err != nil{
			log.Error("file sharding_statedb","func MakeStatedb",err)
			panic(err)
		}
		var rtm RangeRootManage
		err = json.Unmarshal(rm_b,&rtm)
		if err != nil {
			log.Error("file sharding_statedb", "func MakeStatedb:Unmarshal", err)
			panic(err)
		}
		//isex := false
		////判断是否有指定range的rangemanage
		//if rm.Range == b{
		//	isex = true
		//}
		stdb,_ := New(rt,shard.db)
		rm := RangeManage{Range:rtm.Range,State:stdb}

		//如果没有指定range的rangemanage,创建statedb
		//if !isex {
			//通过币种root，创建statedb
			//stdb,_ = New(root,shard.db)
		//}
		//将币种的coinmanage插入shardingdb
	}
	shard.shardings = append(shard.shardings,&CoinManage{Cointyp:cointyp,ValSeg:rms})

}
//func (shard *ShardingStateDB)setError(idx int,err error) {
//	shard.sharding[idx].setError(err)
//
//}

//func (shard *ShardingStateDB)Error(idx int)error  {
//	return shard.sharding[idx].dbErr
//}
//给的根应该是当前分片trie的根hash

func (shard *ShardingStateDB) Reset(root common.Hash) error {
	tr, err := shard.db.OpenTrie(root)
	if err != nil {
		return err
	}
	shard.trie = tr
	shard.shardings = make([]CoinManage,0)
	return nil
}

func (shard *ShardingStateDB) AddLog(log *types.Log) {
	//self:=shard.sharding[idx]
	////slef:=shard.statedb
	//self.journal.append(addLogChange{txhash: self.thash})
	//
	//log.TxHash = self.thash
	//log.BlockHash = self.bhash
	//log.TxIndex = uint(self.txIndex)
	//log.Index = self.logSize
	//self.logs[self.thash] = append(self.logs[self.thash], log)
	//self.logSize++
}

func (shard *ShardingStateDB) GetLogs(hash common.Hash) []*types.Log {
	return nil //shard.sharding[idx].logs[hash]
}

func (shard *ShardingStateDB) Logs() []*types.Log {
	//self:=shard.sharding
	//var logs []*types.Log
	//for i:=0;i<sharding_MOUNTS ;i++  {
	//	for _, lgs := range self[i].logs {
	//		logs = append(logs, lgs...)
	//	}
	//}
	return nil //logs
}

// AddPreimage records a SHA3 preimage seen by the VM.
func (shard *ShardingStateDB) AddPreimage(idx int,hash common.Hash, preimage []byte) {
	//self:=shard.sharding[idx]
	//if _, ok := self.preimages[hash]; !ok {
	//	self.journal.append(addPreimageChange{hash: hash})
	//	pi := make([]byte, len(preimage))
	//	copy(pi, preimage)
	//	self.preimages[hash] = pi
	//}
}

// Preimages returns a list of SHA3 preimages that have been submitted.
func (shard *ShardingStateDB) Preimages(idx int) map[common.Hash][]byte {
	return nil//shard.sharding[idx].preimages
}

func (shard *ShardingStateDB) AddRefund(idx int,gas uint64) {
	//self:=shard.sharding[idx]
	//self.journal.append(refundChange{prev: self.refund})
	//self.refund += gas
}

// Exist reports whether the given account address exists in the state.
// Notably this also returns true for suicided accounts.
func (shard *ShardingStateDB) Exist(addr common.Address) bool {
	return shard.getStateObject(addr) != nil
}



// Empty returns whether the state object is either non-existent
// or empty according to the EIP161 specification (balance = nonce = code = 0)
func (shard *ShardingStateDB) Empty(addr common.Address) bool {
	so := shard.getStateObject(addr)
	return so == nil || so.empty()
}

// Retrieve the balance from the given address or 0 if object not found
func (shard *ShardingStateDB) GetBalance(addr common.Address) common.BalanceType {

	stateObject := shard.getStateObject(addr)
	if stateObject != nil {
		return stateObject.Balance()
	}
	return nil
}

func (shard *ShardingStateDB) GetNonce(addr common.Address) uint64 {
	stateObject := shard.getStateObject(addr)
	if stateObject != nil {
		return stateObject.Nonce()
	}

	return 0 | params.NonceAddOne //YY
}


func (shard *ShardingStateDB) GetCode(idx int,addr common.Address) []byte{

	self:=shard.sharding[idx]
	stateObject := shard.getStateObject(addr)
	if stateObject != nil {
		return stateObject.Code(self.db)
	}
	return nil
}

func (shard *ShardingStateDB) GetCodeSize(idx int,addr common.Address) int {

	self:=shard.sharding[idx]
	stateObject := shard.getStateObject(addr)
	if stateObject == nil {
		return 0
	}
	if stateObject.code != nil {
		return len(stateObject.code)
	}
	size, err := self.db.ContractCodeSize(stateObject.addrHash, common.BytesToHash(stateObject.CodeHash()))
	if err != nil {
		self.setError(err)
	}
	return size
}

func (shard *ShardingStateDB) GetCodeHash(addr common.Address) common.Hash {
	stateObject := shard.getStateObject(addr)
	if stateObject == nil {
		return common.Hash{}
	}
	return common.BytesToHash(stateObject.CodeHash())
}

func (shard *ShardingStateDB) GetState(idx int,addr common.Address, bhash common.Hash) common.Hash {
	stateObject := shard.getStateObject(addr)
	if stateObject != nil {
		return stateObject.GetState(shard.sharding[idx].db, bhash)
	}
	return common.Hash{}
}

func (shard *ShardingStateDB) GetStateByteArray(idx int,a common.Address, b common.Hash) []byte {
	stateObject := shard.getStateObject(a)
	if stateObject != nil {
		return stateObject.GetStateByteArray(shard.sharding[idx].db, b)
	}
	return nil
}

// Database retrieves the low level database supporting the lower level trie ops.
func (shard *ShardingStateDB) Database(idx int) Database {
	return shard.sharding[idx].db
}

// StorageTrie returns the storage trie of an account.
// The return value is a copy and is nil for non-existent accounts.
func (shard *ShardingStateDB) StorageTrie(idx int,addr common.Address) Trie {
	self:=shard.sharding[idx]
	stateObject := self.getStateObject(addr)
	if stateObject == nil {
		return nil
	}
	cpy := stateObject.deepCopy(self)
	return cpy.updateTrie(self.db)
}

func (shard *ShardingStateDB) HasSuicided(idx int,addr common.Address) bool {
	stateObject := shard.sharding[idx].getStateObject(addr)
	if stateObject != nil {
		return stateObject.suicided
	}
	return false
}

/*
 * SETTERS
 */

// AddBalance adds amount to the account associated with addr.
func (shard *ShardingStateDB) AddBalance(accountType uint32, addr common.Address, amount *big.Int) {
	stateObject := shard.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.AddBalance(accountType, amount)
	}
}

// SubBalance subtracts amount from the account associated with addr.
func (shard *ShardingStateDB) SubBalance(accountType uint32, addr common.Address, amount *big.Int) {
	stateObject := shard.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SubBalance(accountType, amount)
	}
}

func (shard *ShardingStateDB) SetBalance(accountType uint32, addr common.Address, amount *big.Int) {
	stateObject := shard.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetBalance(accountType, amount)
	}
}

func (shard *ShardingStateDB) SetNonce(addr common.Address, nonce uint64) {
	stateObject := shard.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetNonce(nonce | params.NonceAddOne) //YY
	}
}

func (shard *ShardingStateDB) SetCode(addr common.Address, code []byte) {
	stateObject := shard.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetCode(crypto.Keccak256Hash(code), code)
	}
}

func (shard *ShardingStateDB) SetState(idx int,addr common.Address, key, value common.Hash) {

	stateObject := shard.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetState(shard.sharding[idx].db, key, value)
	}
}

func (shard *ShardingStateDB) SetStateByteArray(idx int,addr common.Address, key common.Hash, value []byte) {

	stateObject := shard.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetStateByteArray(shard.sharding[idx].db, key, value)
	}
}

// Suicide marks the given account as suicided.
// This clears the account balance.
//
// The account's state object is still available until the state is committed,
// getStateObject will return a non-nil account after Suicide.
func (shard *ShardingStateDB) Suicide(idx int,addr common.Address) bool {
	self:=shard.sharding[idx]
	stateObject := self.getStateObject(addr)
	if stateObject == nil {
		return false
	}
	self.journal.append(suicideChange{
		account: &addr,
		prev:    stateObject.suicided,
		//prevbalance: new(big.Int).Set(stateObject.Balance()),
		prevbalance: stateObject.Balance(),
	})
	stateObject.markSuicided()
	//stateObject.data.Balance = new(big.Int)
	stateObject.data.Balance = make(common.BalanceType, 0)

	return true
}

//
// Setting, updating & deleting state object methods.
//

// updateStateObject writes the given object to the trie.
func (shard *ShardingStateDB) updateStateObject(idx int,stateObject *stateObject) {
	self:=shard.sharding[idx]
	addr := stateObject.Address()
	data, err := rlp.EncodeToBytes(stateObject)
	if err != nil {
		panic(fmt.Errorf("can't encode object at %x: %v", addr[:], err))
	}
	self.setError(self.trie.TryUpdate(addr[:], data))
}

// deleteStateObject removes the given object from the state trie.
func (shard *ShardingStateDB) deleteStateObject(idx int,stateObject *stateObject) {
	self:=shard.sharding[idx]
	stateObject.deleted = true
	addr := stateObject.Address()
	self.setError(self.trie.TryDelete(addr[:]))
}

// Retrieve a state object given by the address. Returns nil if not found.
func (shard ShardingStateDB) getStateObject(idx int,addr common.Address) (stateObject *stateObject) {
	self:=shard.sharding[idx]
	// Prefer 'live' objects.
	if obj := self.stateObjects[addr]; obj != nil {
		if obj.deleted {
			return nil
		}
		return obj
	}

	// Load the object from the database.
	enc, err := self.trie.TryGet(addr[:])
	if len(enc) == 0 {
		self.setError(err)
		return nil
	}
	var data Account
	if err := rlp.DecodeBytes(enc, &data); err != nil {
		log.Error("Failed to decode state object", "addr", addr, "err", err)
		return nil
	}
	// Insert into the live set.
	obj := newObject(self, addr, data)
	self.setStateObject(obj)
	return obj
}

func (shard *ShardingStateDB) setStateObject(idx int,object *stateObject) {
	shard.sharding[idx].stateObjects[object.Address()] = object
}

// Retrieve a state object or create a new state object if nil.
func (shard *ShardingStateDB) GetOrNewStateObject(idx int,addr common.Address) *stateObject {
	self:=shard.sharding[idx]
	stateObject := self.getStateObject(addr)
	if stateObject == nil || stateObject.deleted {
		stateObject, _ = self.createObject(addr)
	}
	return stateObject
}

// createObject creates a new state object. If there is an existing account with
// the given address, it is overwritten and returned as the second return value.
func (shard *ShardingStateDB) createObject(idx int,addr common.Address) (newobj, prev *stateObject) {
	self:=shard.sharding[idx]
	prev = self.getStateObject(addr)
	newobj = newObject(self, addr, Account{})
	newobj.setNonce(0 | params.NonceAddOne) // sets the object to dirty    //YY
	if prev == nil {
		self.journal.append(createObjectChange{account: &addr})
	} else {
		self.journal.append(resetObjectChange{prev: prev})
	}
	self.setStateObject(newobj)
	return newobj, prev
}

// CreateAccount explicitly creates a state object. If a state object with the address
// already exists the balance is carried over to the new account.
//
// CreateAccount is called during the EVM CREATE operation. The situation might arise that
// a contract does the following:
//
//   1. sends funds to sha(account ++ (nonce + 1))
//   2. tx_create(sha(account ++ nonce)) (note that this gets the address of 1)
//
// Carrying over the balance ensures that Maner doesn't disappear.
func (shard *ShardingStateDB) CreateAccount(idx int,addr common.Address) {
	self:=shard.sharding[idx]
	new, prev := self.createObject(addr)
	if prev != nil {
		//new.setBalance(prev.data.Balance)
		for _, tAccount := range prev.data.Balance {
			new.setBalance(tAccount.AccountType, tAccount.Balance)
		}
	}
}

func (shard *ShardingStateDB) ForEachStorage(idx int,addr common.Address, cb func(key, value common.Hash) bool) {
	db:=shard.sharding[idx]
	so := db.getStateObject(addr)
	if so == nil {
		return
	}

	// When iterating over the storage check the cache first
	for h, value := range so.cachedStorage {
		cb(h, value)
	}

	it := trie.NewIterator(so.getTrie(db.db).NodeIterator(nil))
	for it.Next() {
		// ignore cached values
		key := common.BytesToHash(db.trie.GetKey(it.Key))
		if _, ok := so.cachedStorage[key]; !ok {
			cb(key, common.BytesToHash(it.Value))
		}
	}
}

// Copy creates a deep, independent copy of the state.
// Snapshots of the copied state cannot be applied to the copy.
func (shard *ShardingStateDB) Copy(idx int) *StateDB {
	self:=shard.sharding[idx]
	self.lock.Lock()
	defer self.lock.Unlock()

	// Copy all the basic fields, initialize the memory ones
	state := &StateDB{
		db:                self.db,
		trie:              self.db.CopyTrie(self.trie),
		stateObjects:      make(map[common.Address]*stateObject, len(self.journal.dirties)),
		stateObjectsDirty: make(map[common.Address]struct{}, len(self.journal.dirties)),
		refund:            self.refund,
		logs:              make(map[common.Hash][]*types.Log, len(self.logs)),
		logSize:           self.logSize,
		preimages:         make(map[common.Hash][]byte),
		journal:           newJournal(),
	}
	// Copy the dirty states, logs, and preimages
	for addr := range self.journal.dirties {
		// As documented [here](https://github.com/matrix/go-matrix/pull/16485#issuecomment-380438527),
		// and in the Finalise-method, there is a case where an object is in the journal but not
		// in the stateObjects: OOG after touch on ripeMD prior to Byzantium. Thus, we need to check for
		// nil
		if object, exist := self.stateObjects[addr]; exist {
			state.stateObjects[addr] = object.deepCopy(state)
			state.stateObjectsDirty[addr] = struct{}{}
		}
	}
	// Above, we don't copy the actual journal. This means that if the copy is copied, the
	// loop above will be a no-op, since the copy's journal is empty.
	// Thus, here we iterate over stateObjects, to enable copies of copies
	for addr := range self.stateObjectsDirty {
		if _, exist := state.stateObjects[addr]; !exist {
			state.stateObjects[addr] = self.stateObjects[addr].deepCopy(state)
			state.stateObjectsDirty[addr] = struct{}{}
		}
	}

	for hash, logs := range self.logs {
		state.logs[hash] = make([]*types.Log, len(logs))
		copy(state.logs[hash], logs)
	}
	for hash, preimage := range self.preimages {
		state.preimages[hash] = preimage
	}
	return state
}

// Snapshot returns an identifier for the current revision of the state.
func (shard *ShardingStateDB) Snapshot(idx int) int {
	self:=shard.sharding[idx]
	id := self.nextRevisionId
	self.nextRevisionId++
	self.validRevisions = append(self.validRevisions, revision{id, self.journal.length()})
	return id
}

// RevertToSnapshot reverts all state changes made since the given revision.
func (shard *ShardingStateDB) RevertToSnapshot(idx int,revid int) {
	// Find the snapshot in the stack of valid snapshots.
	self:=shard.sharding[idx]
	idx := sort.Search(len(self.validRevisions), func(i int) bool {
		return self.validRevisions[i].id >= revid
	})
	if idx == len(self.validRevisions) || self.validRevisions[idx].id != revid {
		panic(fmt.Errorf("revision id %v cannot be reverted", revid))
	}
	snapshot := self.validRevisions[idx].journalIndex

	// Replay the journal to undo changes and remove invalidated snapshots
	self.journal.revert(self, snapshot)
	self.validRevisions = self.validRevisions[:idx]
}

// GetRefund returns the current value of the refund counter.
func (shard *ShardingStateDB) GetRefund(idx int) uint64 {
	return shard.sharding[idx].refund
}

// Finalise finalises the state by removing the self destructed objects
// and clears the journal as well as the refunds.
func (shard *ShardingStateDB) Finalise(idx int,deleteEmptyObjects bool) {
	s:=shard.sharding[idx]
	for addr := range s.journal.dirties {
		stateObject, exist := s.stateObjects[addr]
		if !exist {
			// ripeMD is 'touched' at block 1714175, in tx 0x1237f737031e40bcde4a8b7e717b2d15e3ecadfe49bb1bbc71ee9deb09c6fcf2
			// That tx goes out of gas, and although the notion of 'touched' does not exist there, the
			// touch-event will still be recorded in the journal. Since ripeMD is a special snowflake,
			// it will persist in the journal even though the journal is reverted. In this special circumstance,
			// it may exist in `s.journal.dirties` but not in `s.stateObjects`.
			// Thus, we can safely ignore it here
			continue
		}

		if stateObject.suicided || (deleteEmptyObjects && stateObject.empty()) {
			s.deleteStateObject(stateObject)
		} else {
			stateObject.updateRoot(s.db)
			s.updateStateObject(stateObject)
		}
		s.stateObjectsDirty[addr] = struct{}{}
	}
	// Invalidate journal because reverting across transactions is not allowed.
	s.clearJournalAndRefund()
}

// IntermediateRoot computes the current root hash of the state trie.
// It is called in between transactions to get the root hash that
// goes into transaction receipts.
func (shard *ShardingStateDB) IntermediateRoot(idx int,deleteEmptyObjects bool) common.Hash {
	s:=shard.sharding[idx]
	s.Finalise(deleteEmptyObjects)
	return s.trie.Hash()
}

// Prepare sets the current transaction hash and index and block hash which is
// used when the EVM emits new state logs.
func (shard *ShardingStateDB) Prepare(idx int,thash, bhash common.Hash, ti int) {
	self:=shard.sharding[idx]
	self.thash = thash
	self.bhash = bhash
	self.txIndex = ti
}

func (shard *ShardingStateDB) clearJournalAndRefund(idx int) {
	s:=shard.sharding[idx]
	s.journal = newJournal()
	s.validRevisions = s.validRevisions[:0]
	s.refund = 0
}

// Commit writes the state to the underlying in-memory trie database.
func (shard *ShardingStateDB) Commit(idx int,deleteEmptyObjects bool) (root common.Hash, err error) {
	s:=shard.sharding[idx]
	defer s.clearJournalAndRefund()

	for addr := range s.journal.dirties {
		s.stateObjectsDirty[addr] = struct{}{}
	}
	// Commit objects to the trie.
	for addr, stateObject := range s.stateObjects {
		_, isDirty := s.stateObjectsDirty[addr]
		switch {
		case stateObject.suicided || (isDirty && deleteEmptyObjects && stateObject.empty()):
			// If the object has been removed, don't bother syncing it
			// and just mark it for deletion in the trie.
			s.deleteStateObject(stateObject)
		case isDirty:
			// Write any contract code associated with the state object
			if stateObject.code != nil && stateObject.dirtyCode {
				s.db.TrieDB().Insert(common.BytesToHash(stateObject.CodeHash()), stateObject.code)
				stateObject.dirtyCode = false
			}
			// Write any storage changes in the state object to its storage trie.
			if err := stateObject.CommitTrie(s.db); err != nil {
				return common.Hash{}, err
			}
			// Update the object in the main account trie.
			s.updateStateObject(stateObject)
		}
		delete(s.stateObjectsDirty, addr)
	}
	// Write trie changes.
	root, err = s.trie.Commit(func(leaf []byte, parent common.Hash) error {
		var account Account
		if err := rlp.DecodeBytes(leaf, &account); err != nil {
			return nil
		}
		if account.Root != emptyState {
			s.db.TrieDB().Reference(account.Root, parent)
		}
		code := common.BytesToHash(account.CodeHash)
		if code != emptyCode {
			s.db.TrieDB().Reference(code, parent)
		}
		return nil
	})
	log.Debug("Trie cache stats after commit", "misses", trie.CacheMisses(), "unloads", trie.CacheUnloads())
	return root, err
}
