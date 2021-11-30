/*
 * Copyright 2021 ICON Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package icsim

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/module"
)

func TestNewJSObject(t *testing.T) {
	intValue := 100
	strValue := "hello"
	bigIntValue := big.NewInt(200)
	addressValue := common.MustNewAddressFromString("hx0")
	addresses := []module.Address{
		common.MustNewAddressFromString("hx1"),
		common.MustNewAddressFromString("hx2"),
		common.MustNewAddressFromString("hx3"),
	}
	bonds := []map[string]interface{}{
		{"address": addresses[0], "value": big.NewInt(0)},
		{"address": addresses[1], "value": big.NewInt(1)},
		{"address": addresses[2], "value": big.NewInt(2)},
	}

	o := make(map[string]interface{})
	o["int"] = intValue
	o["string"] = strValue
	o["bigint"] = bigIntValue
	o["address"] = addressValue
	o["addresses"] = addresses
	o["bonds"] = bonds

	jso := NewJSObject(o)
	assert.Equal(t, jso.Get("int").Int(), intValue)
	assert.Equal(t, jso.Get("string").String(), strValue)
	assert.Zero(t, jso.Get("bigint").BigInt().Cmp(bigIntValue))
	assert.True(t, jso.Get("address").Address().Equal(addressValue))

	assert.Zero(t, jso.Get("bonds").Get(0).Get("value").BigInt().Sign())
	assert.Equal(t, int64(1), GetFromJSOByKeys(o, []interface{}{"bonds", 1, "value"}).(*big.Int).Int64())
}
