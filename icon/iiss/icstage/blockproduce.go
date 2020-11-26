/*
 * Copyright 2020 ICON Foundation
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

package icstage

import (
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icreward"
)

type BlockVotes struct {
	icobject.NoDatabase
	ProposerIndex int
	VoteCount     int
	VoteMask      int64
}

func (bp *BlockVotes) Version() int {
	return 0
}

func (bp *BlockVotes) RLPDecodeFields(decoder codec.Decoder) error {
	_, err := decoder.DecodeMulti(
		&bp.ProposerIndex,
		&bp.VoteCount,
		&bp.VoteMask,
	)
	return err
}

func (bp *BlockVotes) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(
		bp.ProposerIndex,
		bp.VoteCount,
		bp.VoteMask,
	)
}

func (bp *BlockVotes) Equal(o icobject.Impl) bool {
	if bp2, ok := o.(*BlockVotes); ok {
		return bp.ProposerIndex == bp2.ProposerIndex &&
			bp.VoteCount == bp2.VoteCount &&
			bp.VoteMask == bp2.VoteMask
	} else {
		return false
	}
}

func (bp *BlockVotes) Clear() {
	bp.ProposerIndex = 0
	bp.VoteCount = 0
	bp.VoteMask = 0
}

func (bp *BlockVotes) IsEmpty() bool {
	return bp.VoteCount == 0
}

func newBlockVotes(tag icobject.Tag) *BlockVotes {
	return new(BlockVotes)
}

type Validators struct {
	icreward.Validators
}

func (v *Validators) Equal(o icobject.Impl) bool {
	if v2, ok := o.(*Validators); ok {
		if len(v.Addresses) != len(v2.Addresses) {
			return false
		}
		for i, a := range v.Addresses {
			if a.Equal(v2.Addresses[i]) == false {
				return false
			}
		}
		return true
	} else {
		return false
	}
}

func newValidator(tag icobject.Tag) *Validators {
	return new(Validators)
}
