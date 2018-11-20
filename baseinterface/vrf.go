// Copyright (c) 2018 The MATRIX Authors
// Distributed under the MIT software license, see the accompanying
// file COPYING or or http://www.opensource.org/licenses/mit-license.php
package baseinterface

import (
	"crypto/ecdsa"
)

var (
	mapVrf         = make(map[string]func() VrfInterface)
	DefaultVrfPlug = "withHash"
)

//func (self *vrfWithHash)verifyVrf(pk *ecdsa.PublicKey,  prevVrf, newVrf, proof []byte) error {
//func(self *vrfWithHash) computeVrf(sk *ecdsa.PrivateKey,prevVrf []byte) ([]byte, []byte, error) {
type VrfInterface interface {
	ComputeVrf(*ecdsa.PrivateKey, []byte) ([]byte, []byte, error)
	VerifyVrf(*ecdsa.PublicKey, []byte, []byte, []byte) error
}

func NewVrf() VrfInterface {
	return mapVrf[DefaultVrfPlug]()
}

func RegVrf(name string, value func() VrfInterface) {
//	fmt.Println("Vrf插件 注册函数", "name", name)
	mapVrf[name] = value
}
