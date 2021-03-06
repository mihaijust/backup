// Copyright (c) 2018 The MATRIX Authors
// Distributed under the MIT software license, see the accompanying
// file COPYING or or http://www.opensource.org/licenses/mit-license.php
package metrics

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestRegistryMarshallJSON(t *testing.T) {
	b := &bytes.Buffer{}
	enc := json.NewEncoder(b)
	r := NewRegistry()
	r.Register("counter", NewCounter())
	enc.Encode(r)
	if s := b.String(); "{\"counter\":{\"count\":0}}\n" != s {
		t.Fatalf(s)
	}
}

func TestRegistryWriteJSONOnce(t *testing.T) {
	r := NewRegistry()
	r.Register("counter", NewCounter())
	b := &bytes.Buffer{}
	WriteJSONOnce(r, b)
	if s := b.String(); s != "{\"counter\":{\"count\":0}}\n" {
		t.Fail()
	}
}
