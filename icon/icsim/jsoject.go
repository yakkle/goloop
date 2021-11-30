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

	"github.com/icon-project/goloop/module"
)

type JSObject struct {
	o interface{}
}

func (jso *JSObject) Int() int {
	return jso.o.(int)
}

func (jso *JSObject) BigInt() *big.Int {
	return jso.o.(*big.Int)
}

func (jso *JSObject) String() string {
	return jso.o.(string)
}

func (jso *JSObject) Address() module.Address {
	return jso.o.(module.Address)
}

func (jso *JSObject) Get(key interface{}) *JSObject {
	var o interface{}
	switch key.(type) {
	case int:
		i := key.(int)
		if l, ok := jso.o.([]interface{}); ok {
			o = l[i]
		} else {
			o = jso.o.([]map[string]interface{})[i]
		}
	case string:
		o = jso.o.(map[string]interface{})[key.(string)]
	default:
		return nil
	}
	return NewJSObject(o)
}

func GetFromJSO(o interface{}, key interface{}) interface{} {
	keys := []interface{}{key}
	return GetFromJSOByKeys(o, keys)
}

func GetFromJSOByKeys(o interface{}, keys []interface{}) interface{} {
	for _, key := range keys {
		switch key.(type) {
		case int:
			i := key.(int)
			if l, ok := o.([]interface{}); ok {
				o = l[i]
			} else {
				o = o.([]map[string]interface{})[i]
			}
		case string:
			o = o.(map[string]interface{})[key.(string)]
		default:
			return nil
		}
	}
	return o
}

func NewJSObject(o interface{}) *JSObject {
	return &JSObject{o: o}
}
