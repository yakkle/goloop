package icsim

/*
func TestSimulator_UnbondOnRev13(t *testing.T) {
	const (
		termPeriod                           = 100
		mainPRepCount                        = 22
		validationPenaltyCondition           = 5
		consistentValidationPenaltyCondition = 3
	)

	var err error
	var csi module.ConsensusInfo
	var vl []module.Validator
	//var prep *icstate.PRep
	var receipts []Receipt
	//var oldBonded, bonded, slashed *big.Int
	var tx Transaction
	var voted []bool
	var slashRatio = 0
	var env *Env

	c := NewConfig()
	c.MainPRepCount = mainPRepCount
	c.TermPeriod = termPeriod
	c.ValidationPenaltyCondition = validationPenaltyCondition
	c.ConsistentValidationPenaltyCondition = consistentValidationPenaltyCondition
	c.ConsistentValidationPenaltySlashRatio = slashRatio

	voted = make([]bool, mainPRepCount)
	for i := 0; i < len(voted); i++ {
		voted[i] = true
	}

	// Decentralization is activated
	env = initEnv(t, c, icmodule.Revision13)
	sim := env.sim
	assertBondsOfPReps(t, sim, env.preps)

	// Unbonding
	vl = sim.ValidatorList()
	csi = newConsensusInfo(sim.Database(), vl, voted)
	bonder := env.bonders[0]
	bonds := make(icstate.Bonds, 0)
	tx = sim.SetBond(bonder, bonds)
	_, err = sim.GoByTransaction(tx, csi)
	assert.NoError(t, err)
	assertBondsOfPReps(t, sim, env.preps)
	assertBondsOfUser(t, sim, bonder)

	// Fails to remove an address which have unbondings from bonderList
	vl = sim.ValidatorList()
	csi = newConsensusInfo(sim.Database(), vl, voted)
	bonderList := make(icstate.BonderList, 1)
	bonderList[0] = common.AddressToPtr(env.god)
	tx = sim.SetBonderList(env.preps[0], bonderList)
	receipts, err = sim.GoByTransaction(tx, nil)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(receipts))
	assert.Zero(t, receipts[0].Status())

	// Succeeds to add another address to bonderList without removing a existing bonder
	bonderList = icstate.BonderList{
		common.AddressToPtr(env.god),
		common.AddressToPtr(env.bonders[0]),
	}
	tx = sim.SetBonderList(env.preps[0], bonderList)
	receipts, err = sim.GoByTransaction(tx, nil)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(receipts))
	assert.Equal(t, 1, receipts[0].Status())

	jso := sim.GetBond(bonder)
	expireBlockHeight := GetFromJSOByKeys(jso, []interface{}{"unbonds", 0, "expireBlockHeight"}).(int64)
	assert.True(t, expireBlockHeight > sim.BlockHeight())

	err = sim.GoTo(expireBlockHeight, nil)
	assert.Equal(t, expireBlockHeight, sim.BlockHeight())
	assert.NoError(t, err)
	jso = sim.GetBond(bonder)
	assert.Equal(t, 1, len(GetFromJSO(jso, "unbonds").([]interface{})))

	err = sim.Go(1, nil)
	assert.NoError(t, err)
	jso = sim.GetBond(bonder)
	assert.Equal(t, 0, len(GetFromJSO(jso, "unbonds").([]interface{})))
}
*/
