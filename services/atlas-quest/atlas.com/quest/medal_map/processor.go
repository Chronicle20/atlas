package medal_map

import (
	"context"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Processor records a character's visited maps for a medal-style quest and
// reports the resulting distinct-map count (task-290 G14 / explorer_quest).
//
// Cosmic's explorerQuest also compares that count against the quest's
// infoEx(0) threshold to decide between the completion packet and the
// "<n>/<m> regions explored" title message. Neither the quest's per-status
// infoNumber nor infoEx is served by atlas-data (grep of
// services/atlas-data/atlas.com/data/quest/reader.go found only the
// Check.img start/end-requirement infoNumber, never infoEx), so that
// comparison cannot be made faithfully here. Record records the count and
// nothing else; the completion/threshold decision is out of scope until a
// data source exists.
type Processor interface {
	Record(characterId uint32, questId uint32, mapId _map.Id) (RecordResult, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   tenant.MustFromContext(ctx),
	}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) Record(characterId uint32, questId uint32, mapId _map.Id) (RecordResult, error) {
	db := p.db.WithContext(p.ctx)

	newly, err := recordIfAbsent(db, p.t.Id(), characterId, questId, uint32(mapId))
	if err != nil {
		p.l.WithError(err).Errorf("Unable to record medal map [%d] for character [%d] quest [%d].", mapId, characterId, questId)
		return RecordResult{}, err
	}

	count, err := countByCharacterAndQuest(db, characterId, questId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to count medal maps for character [%d] quest [%d].", characterId, questId)
		return RecordResult{}, err
	}

	if newly {
		p.l.Debugf("Recorded medal map [%d] for character [%d] quest [%d]; count now [%d].", mapId, characterId, questId, count)
	} else {
		p.l.Debugf("Medal map [%d] already recorded for character [%d] quest [%d]; count [%d].", mapId, characterId, questId, count)
	}

	return RecordResult{Count: count, NewlyRecorded: newly}, nil
}
