/*
 * Copyright 2020 ICON Foundation
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package iiss

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/icon/iiss/icreward"
	"github.com/icon-project/goloop/icon/iiss/icstage"
	"github.com/icon-project/goloop/icon/iiss/icstate"
)

func MakeCalculator(database db.Database, back *icstage.Snapshot) *Calculator {
	c := new(Calculator)
	c.back = back
	c.temp = icreward.NewState(database, nil)

	return c
}

func TestCalculator_processClaim(t *testing.T) {
	database := db.NewMapDB()
	s := icstage.NewState(database, nil)

	addr1 := common.NewAddressFromString("hx1")
	addr2 := common.NewAddressFromString("hx2")
	v1 := int64(100)
	v2 := int64(200)

	type args struct {
		addr  *common.Address
		value *big.Int
	}

	tests := []struct {
		name string
		args args
		want int64
	}{
		{
			"Add Claim 100",
			args{
				addr1,
				big.NewInt(v1),
			},
			v1,
		},
		{
			"Add Claim 200 to new address",
			args{
				addr2,
				big.NewInt(v2),
			},
			v2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := tt.args
			err := s.AddIScoreClaim(args.addr, args.value)
			assert.NoError(t, err)
		})
	}

	c := MakeCalculator(database, s.GetSnapshot())

	err := c.processClaim()
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := tt.args
			iScore, err := c.temp.GetIScore(args.addr)
			assert.NoError(t, err)
			assert.Equal(t, 0, args.value.Cmp(iScore.Value))
		})
	}
}

func TestCalculator_processBlockProduce(t *testing.T) {
	database := db.NewMapDB()
	s := icstage.NewState(database, nil)

	offset1 := 0
	offset2 := 5

	addr1 := common.NewAddressFromString("hx1")
	addr2 := common.NewAddressFromString("hx2")
	addr3 := common.NewAddressFromString("hx3")
	addr4 := common.NewAddressFromString("hx4")
	addr5 := common.NewAddressFromString("hx5")

	type args struct {
		type_        int
		offset       int
		proposeIndex int
		voteCount    int
		voteMask     int64
		validators   []*common.Address
	}

	tests := []struct {
		name string
		args args
	}{
		{
			"Validator 1",
			args{
				type_:      icstage.TypeValidator,
				offset:     offset1,
				validators: []*common.Address{addr1, addr2, addr3, addr4},
			},
		},
		{
			"block produce 1",
			args{
				type_:        icstage.TypeBlockProduce,
				offset:       offset1,
				proposeIndex: 1,
				voteCount:    4,
				voteMask:     0b1111,
			},
		},
		{
			"Validator 2",
			args{
				type_:      icstage.TypeValidator,
				offset:     offset2,
				validators: []*common.Address{addr1, addr2, addr3, addr5},
			},
		},
		{
			"block produce 2",
			args{
				type_:        icstage.TypeBlockProduce,
				offset:       offset2,
				proposeIndex: 3,
				voteCount:    3,
				voteMask:     0b1110,
			},
		},
	}
	for _, tt := range tests {
		args := tt.args
		t.Run(tt.name, func(t *testing.T) {
			switch args.type_ {
			case icstage.TypeBlockProduce:
				err := s.AddBlockVotes(args.offset, args.proposeIndex, args.voteCount, args.voteMask)
				assert.NoError(t, err)
			case icstage.TypeValidator:
				err := s.AddValidators(args.offset, args.validators)
				assert.NoError(t, err)
			}
		})
	}

	c := MakeCalculator(database, s.GetSnapshot())
	irep := big.NewInt(int64(YearBlock * IScoreICXRatio))
	vs := make([]*validator, 0)
	var err error

	for offset := offset1; offset <= offset2; offset += 1 {
		vs, err = c.processBlockProduce(irep, offset, vs)
		assert.NoError(t, err)
	}
	// Beta1 in temp made by tests[0] and tests[1]
	rewardGenerate := new(big.Int).Div(irep, bigIntBeta1Divider)
	rewardValidate := new(big.Int).Div(irep, bigIntBeta1Divider)
	for i, v := range tests[0].args.validators {
		is, err := c.temp.GetIScore(v)
		assert.NoError(t, err)
		reward := new(big.Int)
		if i == tests[1].args.proposeIndex {
			reward.Add(reward, rewardGenerate)
		}
		if (tests[1].args.voteMask & (1 << i)) != 0 {
			r := new(big.Int).Div(rewardValidate, big.NewInt(int64(tests[1].args.voteCount)))
			reward.Add(reward, r)
		}
		assert.Equal(t, reward.Int64(), is.Value.Int64())
	}

	// Beta1 in validator list make by tests[2] and tests[3]
	assert.Equal(t, len(tests[2].args.validators), len(vs))
	for i, v := range vs {
		reward := new(big.Int)
		if i == tests[3].args.proposeIndex {
			reward.Add(reward, rewardGenerate)
		}
		if (tests[3].args.voteMask & (1 << i)) != 0 {
			r := new(big.Int).Div(rewardValidate, big.NewInt(int64(tests[3].args.voteCount)))
			reward.Add(reward, r)
		}
		assert.Equal(t, 0, v.iScore.Cmp(reward))
	}
}

func newDelegatedDataForTest(enable bool, current int64, snapshot int64, iScore int64) *delegatedData {
	return &delegatedData{
		delegated: &icreward.Delegated{
			Enable:   enable,
			Current:  big.NewInt(current),
			Snapshot: big.NewInt(snapshot),
		},
		iScore: big.NewInt(iScore),
	}
}

func TestDelegatedData_compare(t *testing.T) {
	d1 := newDelegatedDataForTest(true, 10, 10, 10)
	d2 := newDelegatedDataForTest(true, 20, 20, 20)
	d3 := newDelegatedDataForTest(true, 21, 20, 21)
	d4 := newDelegatedDataForTest(false, 30, 30, 30)
	d5 := newDelegatedDataForTest(false, 31, 30, 31)

	type args struct {
		d1 *delegatedData
		d2 *delegatedData
	}

	tests := []struct {
		name string
		args args
		want int
	}{
		{
			"x<y",
			args{d1, d2},
			-1,
		},
		{
			"x<y,disable",
			args{d5, d2},
			-1,
		},
		{
			"x==y",
			args{d2, d3},
			0,
		},
		{
			"x==y,disable",
			args{d4, d5},
			0,
		},
		{
			"x>y",
			args{d3, d1},
			1,
		},
		{
			"x>y,disable",
			args{d1, d4},
			1,
		},
	}
	for _, tt := range tests {
		args := tt.args
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, args.d1.compare(args.d2))
		})
	}
}

func TestDelegated_setEnable(t *testing.T) {
	d := newDelegated()
	for i := int64(1); i < 6; i += 1 {
		addr := common.NewAddressFromString(fmt.Sprintf("hx%d", i))
		data := newDelegatedDataForTest(true, i, i, i)
		d.addDelegatedData(addr, data)
	}

	enable := true
	for a, dd := range d.preps {
		enable = !enable
		d.setEnable(&a, enable)
		assert.Equal(t, enable, dd.delegated.Enable)
	}

	newAddr := common.NewAddressFromString("hx123412341234")
	d.setEnable(newAddr, true)
	assert.Equal(t, true, d.preps[*newAddr].delegated.Enable)
	assert.True(t, d.preps[*newAddr].delegated.IsEmpty())
	assert.Equal(t, 0, d.preps[*newAddr].iScore.Sign())
}

func TestDelegated_updateCurrent(t *testing.T) {
	d := newDelegated()
	ds := make([]*icstate.Delegation, 0)
	for i := int64(1); i < 6; i += 1 {
		addr := common.NewAddressFromString(fmt.Sprintf("hx%d", i))
		data := newDelegatedDataForTest(true, i, i, i)
		d.addDelegatedData(addr, data)

		ds = append(
			ds,
			&icstate.Delegation{
				Address: addr,
				Value:   common.NewHexInt(i),
			},
		)
	}
	newAddr := common.NewAddressFromString("hx321321")
	ds = append(
		ds,
		&icstate.Delegation{
			Address: newAddr,
			Value:   common.NewHexInt(100),
		},
	)

	d.updateCurrent(ds)
	for _, v := range ds {
		expect := v.Value.Value().Int64() * 2
		if v.Address.Equal(newAddr) {
			expect = v.Value.Value().Int64()
		}
		assert.Equal(t, expect, d.preps[*v.Address].delegated.Current.Int64())
	}
}

func TestDelegated_updateSnapshot(t *testing.T) {
	d := newDelegated()
	for i := int64(1); i < 6; i += 1 {
		addr := common.NewAddressFromString(fmt.Sprintf("hx%d", i))
		data := newDelegatedDataForTest(true, i*2, i, i)
		d.addDelegatedData(addr, data)
	}

	d.updateSnapshot()

	for _, prep := range d.preps {
		assert.Equal(t, 0, prep.delegated.Current.Cmp(prep.delegated.Snapshot))
	}
}

func TestDelegated_updateTotal(t *testing.T) {
	d := newDelegated()
	total := int64(0)
	more := int64(10)
	maxIndex := int64(d.maxRankForReward()) + more
	for i := int64(1); i <= maxIndex; i += 1 {
		addr := common.NewAddressFromString(fmt.Sprintf("hx%d", i))
		data := newDelegatedDataForTest(true, i, i, i)
		d.addDelegatedData(addr, data)
		if i > more {
			total += i
		}
	}
	d.updateTotal()
	assert.Equal(t, total, d.total.Int64())

	for i, rank := range d.rank {
		expect := common.NewAddressFromString(fmt.Sprintf("hx%d", maxIndex-int64(i)))
		assert.True(t, expect.Equal(&rank))
	}
}

func TestDelegated_calculateReward(t *testing.T) {
	d := newDelegated()
	total := int64(0)
	more := int64(10)
	maxIndex := int64(d.maxRankForReward()) + more
	for i := int64(1); i <= maxIndex; i += 1 {
		addr := common.NewAddressFromString(fmt.Sprintf("hx%d", i))
		data := newDelegatedDataForTest(true, i, i, 0)
		d.addDelegatedData(addr, data)
		if i > more {
			total += i
		}
	}
	d.updateTotal()
	assert.Equal(t, total, d.total.Int64())

	irep := big.NewInt(int64(YearBlock))
	period := MonthBlock
	bigIntPeriod := big.NewInt(int64(period))

	d.calculateReward(irep, period)

	for i, addr := range d.rank {
		expect := big.NewInt(maxIndex - int64(i))
		if i >= d.maxRankForReward() {
			expect.SetInt64(0)
		} else {
			expect.Mul(expect, irep)
			expect.Mul(expect, bigIntPeriod)
			expect.Div(expect, bigIntBeta2Divider)
			expect.Div(expect, d.total)
		}
		assert.Equal(t, expect.Int64(), d.preps[addr].iScore.Int64(), i)
	}
}

func TestCalculator_DelegatingReward(t *testing.T) {
	addr1 := common.NewAddressFromString("hx1")
	addr2 := common.NewAddressFromString("hx2")
	addr3 := common.NewAddressFromString("hx3")
	addr4 := common.NewAddressFromString("hx4")
	prepInfo := map[common.Address]*pRepEnable{
		*addr1: {0, 0},
		*addr2: {10, 0},
		*addr3: {100, 200},
	}

	d1 := &icstate.Delegation{
		Address: addr1,
		Value:   common.NewHexInt(100),
	}
	d2 := &icstate.Delegation{
		Address: addr2,
		Value:   common.NewHexInt(100),
	}
	d3 := &icstate.Delegation{
		Address: addr3,
		Value:   common.NewHexInt(100),
	}
	d4 := &icstate.Delegation{
		Address: addr4,
		Value:   common.NewHexInt(100),
	}

	type args struct {
		rrep       int
		from       int
		to         int
		delegating icstate.Delegations
	}

	tests := []struct {
		name string
		args args
		want int64
	}{
		{
			name: "PRep-full",
			args: args{
				100,
				0,
				1000,
				icstate.Delegations{d1},
			},
			want: 100 * 100 * 1000 * 1000 / YearBlock,
		},
		{
			name: "PRep-enabled",
			args: args{
				100,
				0,
				1000,
				icstate.Delegations{d2},
			},
			want: 100 * 100 * (1000 - 10) * 1000 / YearBlock,
		},
		{
			name: "PRep-disabled",
			args: args{
				100,
				0,
				1000,
				icstate.Delegations{d3},
			},
			want: 100 * 100 * (200 - 100) * 1000 / YearBlock,
		},
		{
			name: "PRep-None",
			args: args{
				100,
				0,
				1000,
				icstate.Delegations{d4},
			},
			want: 0,
		},
		{
			name: "PRep-combination",
			args: args{
				100,
				0,
				1000,
				icstate.Delegations{d1, d2, d3, d4},
			},
			want: (100 * 100 * 1000 * 1000 / YearBlock) +
				(100 * 100 * (1000 - 10) * 1000 / YearBlock) +
				(100 * 100 * (200 - 100) * 1000 / YearBlock),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := tt.args
			reward := delegatingReward(
				big.NewInt(int64(args.rrep)),
				args.from,
				args.to,
				prepInfo,
				args.delegating,
			)
			assert.Equal(t, tt.want, reward.Int64())
		})
	}
}

func TestCalculator_processDelegating(t *testing.T) {
	database := db.NewMapDB()
	s := icstage.NewState(database, nil)
	c := MakeCalculator(database, s.GetSnapshot())

	rrep := 100
	rrepBigInt := big.NewInt(100)
	from := 0
	to := 100
	offset := 50

	addr1 := common.NewAddressFromString("hx1")
	addr2 := common.NewAddressFromString("hx2")
	addr3 := common.NewAddressFromString("hx3")
	addr4 := common.NewAddressFromString("hx4")

	d1Value := 100
	d2Value := 200
	d1 := &icstate.Delegation{
		Address: addr1,
		Value:   common.NewHexInt(int64(d1Value)),
	}
	d2 := &icstate.Delegation{
		Address: addr1,
		Value:   common.NewHexInt(int64(d2Value)),
	}
	ds1 := icstate.Delegations{d1}
	ds2 := icstate.Delegations{d2}

	// make pRepInfo. all enabled
	prepInfo := make(map[common.Address]*pRepEnable)
	prepInfo[*addr1] = &pRepEnable{0, 0}

	// write delegating data to base
	dting1 := icreward.NewDelegating()
	dting1.Delegations = ds1
	dting2 := icreward.NewDelegating()
	dting2.Delegations = ds2
	//c.temp.SetDelegating(addr2, dting2)
	//c.temp.SetDelegating(addr3, dting1)
	//c.temp.SetDelegating(addr4, dting2)
	c.temp.SetDelegating(addr2, dting2.Clone())
	c.temp.SetDelegating(addr3, dting1.Clone())
	c.temp.SetDelegating(addr4, dting2.Clone())
	c.base = c.temp.GetSnapshot()

	// make delegationMap
	delegationMap := make(map[common.Address]map[int]icstate.Delegations)
	delegationMap[*addr1] = make(map[int]icstate.Delegations)
	delegationMap[*addr1][from+offset] = ds2
	delegationMap[*addr3] = make(map[int]icstate.Delegations)
	delegationMap[*addr3][from+offset] = ds2
	delegationMap[*addr4] = make(map[int]icstate.Delegations)
	delegationMap[*addr4][from+offset] = icstate.Delegations{}

	err := c.processDelegating(rrepBigInt, from, to, prepInfo, delegationMap)
	assert.NoError(t, err)

	type args struct {
		addr *common.Address
	}

	tests := []struct {
		name       string
		args       args
		want       int64
		delegating *icreward.Delegating
	}{
		{
			name:       "Delegate New",
			args:       args{addr1},
			want:       int64(rrep * d2Value * (to - offset) * IScoreICXRatio / YearBlock),
			delegating: dting2,
		},
		{
			name:       "Delegated and no modification",
			args:       args{addr2},
			want:       int64(rrep * d2Value * (to - from) * IScoreICXRatio / YearBlock),
			delegating: dting2,
		},
		{
			name:       "Delegated and modified",
			args:       args{addr3},
			want:       int64(rrep*d1Value*(offset-from)*IScoreICXRatio/YearBlock) + int64(rrep*d2Value*(to-offset)*IScoreICXRatio/YearBlock),
			delegating: dting2,
		},
		{
			name:       "Delegating removed",
			args:       args{addr4},
			want:       int64(rrep * d2Value * (offset - from) * IScoreICXRatio / YearBlock),
			delegating: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := tt.args

			iScore, err := c.temp.GetIScore(args.addr)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, iScore.Value.Int64())

			delegating, err := c.temp.GetDelegating(args.addr)
			assert.NoError(t, err)
			if tt.delegating != nil {
				assert.NotNil(t, delegating)
				assert.True(t, delegating.Equal(tt.delegating))
			} else {
				assert.Nil(t, delegating)
			}
		})
	}
}
