package sync2

import (
	"context"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/icon-project/goloop/btp"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/txresult"
)

const (
	protoV1  byte = 1
	protoV2  byte = 2
	protoAny byte = protoV1 | protoV2
)

type syncer struct {
	logger log.Logger

	database   db.Database
	plt        Platform
	reactors   []SyncReactor
	processors []SyncProcessor
	noBuffer   bool

	ah  []byte // account hash
	vlh []byte // validator list hash
	ed  []byte // extension data
	prh []byte // patch receipt hash
	nrh []byte // normal receipt hash
	bh  []byte // btp hash

	// Sync Result
	wss state.WorldSnapshot
	prl module.ReceiptList
	nrl module.ReceiptList
	bd  module.BTPDigest
}

func (s *syncer) newMerkleBuilder() merkle.Builder {
	if s.noBuffer {
		return merkle.NewBuilderWithRawDatabase(s.database)
	} else {
		return merkle.NewBuilder(s.database)
	}
}

func (s *syncer) getStateBuilder(accountsHash, pReceiptsHash, nReceiptsHash, validatorListHash, extensionData []byte) merkle.Builder {
	s.logger.Debugf("GetStateBuilder ah=%#x, prh=%#x, nrh=%#x, vlh=%#x, ed=%#x",
		accountsHash, pReceiptsHash, nReceiptsHash, validatorListHash, extensionData)
	builder := s.newMerkleBuilder()
	ess := s.plt.NewExtensionWithBuilder(builder, extensionData)

	if wss, err := state.NewWorldSnapshotWithBuilder(builder, accountsHash, validatorListHash, ess, nil); err == nil {
		s.wss = wss
	}

	s.prl = txresult.NewReceiptListWithBuilder(builder, pReceiptsHash)
	s.nrl = txresult.NewReceiptListWithBuilder(builder, nReceiptsHash)

	return builder
}

func (s *syncer) getBTPBuilder(btpHash []byte) merkle.Builder {
	s.logger.Debugf("GetBTPBuilder bh=%#x", btpHash)
	if len(btpHash) == 0 {
		s.bd = btp.ZeroDigest
		return nil
	}
	builder := s.newMerkleBuilder()

	btpDigest, err := btp.NewDigestWithBuilder(builder, btpHash)
	if err == nil {
		s.bd = btpDigest
	} else {
		s.logger.Errorf("Failed NewDigestWithBuilder. err=%+v", err)
		return nil
	}

	return builder
}

func timeElapsed(name string, logger log.Logger) func() {
	logger.Infof("%s start", name)
	start := time.Now()
	return func() {
		logger.Infof("%s elapsed=%v", name, time.Since(start))
	}
}

// syncWithBuilders start Sync
func (s *syncer) syncWithBuilders(stateBuilders []merkle.Builder, btpBuilders []merkle.Builder) (*Result, error) {
	s.logger.Debugln("SyncWithBuilders()")
	egrp, _ := errgroup.WithContext(context.Background())

	for _, builder := range stateBuilders {
		// sync processor with v1,v2 protocol
		sp := newSyncProcessor(builder, s.reactors, s.logger, false)
		egrp.Go(sp.DoSync)
		s.processors = append(s.processors, sp)
	}

	var reactorsV2 []SyncReactor
	for _, reactor := range s.reactors {
		if reactor.GetVersion() == protoV2 {
			reactorsV2 = append(reactorsV2, reactor)
		}
	}

	for _, builder := range btpBuilders {
		// sync processor with v2 protocol
		sp := newSyncProcessor(builder, reactorsV2, s.logger, false)
		egrp.Go(sp.DoSync)
		s.processors = append(s.processors, sp)
	}

	if err := egrp.Wait(); err != nil {
		return nil, err
	}

	result := &Result{
		s.wss, s.prl, s.nrl, s.bd,
	}
	s.logger.Debugln("SyncWithBuilders() done!")
	return result, nil
}

func (s *syncer) ForceSync() (*Result, error) {
	defer timeElapsed("ForceSync", s.logger)()
	var stateBuilders, btpBuilders []merkle.Builder

	stateBuilder := s.getStateBuilder(s.ah, s.prh, s.nrh, s.vlh, s.ed)
	stateBuilders = append(stateBuilders, stateBuilder)

	btpBuilder := s.getBTPBuilder(s.bh)
	if btpBuilder != nil {
		btpBuilders = append(btpBuilders, btpBuilder)
	}

	return s.syncWithBuilders(stateBuilders, btpBuilders)
}

// Stop sync
func (s *syncer) Stop() {
	for _, sp := range s.processors {
		sp.Stop()
	}
}

// Finalize Sync
func (s *syncer) Finalize() error {
	s.logger.Debugf("Finalize : ah=%#x, prh=%#x, nrh=%#x, vlh=%#x, ed=%#x, bh=%#x",
		s.ah, s.prh, s.nrh, s.vlh, s.ed, s.bh)

	for i, sp := range s.processors {
		sproc := sp.(*syncProcessor)
		if sproc.builder == nil {
			continue
		} else {
			s.logger.Tracef("Flush syncprocessor=%v", sp)
			if err := sproc.builder.Flush(true); err != nil {
				s.logger.Errorf("Failed to flush for %d builder err=%+v", i, err)
				return err
			}
		}
	}

	s.processors = make([]SyncProcessor, 0)
	return nil
}

func newSyncerWithHashes(database db.Database, reactors []SyncReactor, plt Platform,
	ah, prh, nrh, vlh, ed, bh []byte, logger log.Logger, noBuffer bool) Syncer {
	s := &syncer{
		logger:   logger,
		database: database,
		noBuffer: noBuffer,
		reactors: reactors,
		plt:      plt,
		ah:       ah,
		vlh:      vlh,
		prh:      prh,
		nrh:      nrh,
		ed:       ed,
		bh:       bh,
	}

	return s
}
