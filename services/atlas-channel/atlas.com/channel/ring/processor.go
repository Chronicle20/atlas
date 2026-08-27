package ring

import (
	"atlas-channel/equipment"
	"context"

	"github.com/sirupsen/logrus"

	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"

	slot2 "github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// ringSlotPositions are the four couple/friendship ring sub-slots
// (libs/atlas-constants/inventory/slot/constants.go: ring1=-12, ring2=-13,
// ring3=-15, ring4=-16). The petRing*/pet2Ring*/pet3Ring* positions
// (-21, -29, -31, -37, -39, -45) are pet equipment, not couple/friendship
// rings, and are deliberately excluded.
var ringSlotPositions = map[slot2.Position]bool{
	-12: true,
	-13: true,
	-15: true,
	-16: true,
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

type Processor interface {
	// GetRingSet returns the couple/friendship ring pair currently equipped
	// by characterId, joined from the cached ring pair halves (populated by
	// Populate) against eq's equipped cash assets. It never issues a REST
	// call -- population happens once at character load (PRD §8), not on
	// encode -- so a cache miss returns an empty RingSet and logs at debug
	// rather than blocking the caller to fetch.
	//
	// Selection rule, per ring type (COUPLE, FRIENDSHIP): among the
	// character's ACTIVE halves of that type whose cash id is currently
	// equipped in one of the four ring sub-slots (ring1..ring4, positions
	// -12/-13/-15/-16), the half in the numerically HIGHEST (least
	// negative -- ring1 before ring2 before ring3 before ring4) slot
	// position wins; a tie is broken by the lower cash id. BROKEN/EXPIRED
	// halves and halves equipped in a non-cash sub-slot never match. The
	// Marriage arm is always nil: PRD §2 lists marriage-ring acquisition as
	// a non-goal, and atlas-cashshop's ring.Type only admits COUPLE and
	// FRIENDSHIP.
	GetRingSet(characterId uint32, eq equipment.Model) packetmodel.RingSet

	// GetRingRecords returns the couple/friendship ring record block for
	// characterId (CharacterData's site A, Task 3), joined from the same
	// cached ring pair halves as GetRingSet. Unlike GetRingSet's
	// currently-equipped selection, this is a history view: every ACTIVE
	// half the character owns is listed, whether or not it is currently
	// equipped in a ring sub-slot. It never issues a REST call, for the
	// same reason as GetRingSet. The Marriage arm is always empty: see
	// GetRingSet's doc comment.
	GetRingRecords(characterId uint32) packetmodel.RingRecords

	// Invalidate drops the cached ring pair halves for characterId. It does
	// not refetch; the next Populate call repopulates.
	Invalidate(characterId uint32)

	// Populate fetches every ring pair half owned by characterId from
	// atlas-cashshop and caches it. This is the fail-soft REST entry point
	// (PRD FR-5): a cashshop outage degrades to no cached halves --
	// GetRingSet then returns an empty RingSet -- rather than failing
	// character spawn. Populate is idempotent while the character stays
	// cached: a call that finds an existing entry (however it got there --
	// including an earlier zero-halves population) returns immediately
	// without issuing a REST call, so a caller on the character-load path
	// (login/channel-enter) may call it once per load without worrying
	// about a duplicate delivery double-fetching. Invalidate clears the
	// entry so the next Populate call actually re-fetches.
	Populate(characterId uint32) error
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

// upstreamFn is the test-overridable upstream fetch (monster/information's
// upstreamFn precedent). It drains every page of atlas-cashshop's
// GET /rings?filter[characterId] list (requests.DrainProvider, since the
// list is server-side paginated -- task-269 task 8, mirroring task-117's
// door list).
var upstreamFn = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) ([]Model, error) {
	url, err := requestByCharacterId(ctx, characterId)
	if err != nil {
		return nil, err
	}
	return requests.DrainProvider[RestModel, Model](l, ctx)(url, 250, Extract, model.Filters[Model]())()
}

func (p *ProcessorImpl) Populate(characterId uint32) error {
	t := tenant.MustFromContext(p.ctx)
	if _, ok := getRingCache().lookup(t.Id(), characterId); ok {
		return nil
	}

	halves, err := upstreamFn(p.l, p.ctx, characterId)
	if err != nil {
		p.l.WithError(err).Warnf("Unable to retrieve ring pairs for character [%d]; ring set will be empty until population succeeds.", characterId)
		return nil
	}
	getRingCache().put(t.Id(), characterId, cacheEntry{halves: halves})
	return nil
}

func (p *ProcessorImpl) Invalidate(characterId uint32) {
	t := tenant.MustFromContext(p.ctx)
	getRingCache().invalidate(t.Id(), characterId)
}

func (p *ProcessorImpl) GetRingSet(characterId uint32, eq equipment.Model) packetmodel.RingSet {
	t := tenant.MustFromContext(p.ctx)
	e, ok := getRingCache().lookup(t.Id(), characterId)
	if !ok {
		p.l.Debugf("No cached ring pairs for character [%d]; encoding empty ring set.", characterId)
		return packetmodel.RingSet{}
	}
	return packetmodel.RingSet{
		Couple:     selectPair(e.halves, TypeCouple, eq),
		Friendship: selectPair(e.halves, TypeFriendship, eq),
	}
}

func (p *ProcessorImpl) GetRingRecords(characterId uint32) packetmodel.RingRecords {
	t := tenant.MustFromContext(p.ctx)
	e, ok := getRingCache().lookup(t.Id(), characterId)
	if !ok {
		p.l.Debugf("No cached ring pairs for character [%d]; encoding empty ring records.", characterId)
		return packetmodel.RingRecords{}
	}

	var rr packetmodel.RingRecords
	for _, h := range e.halves {
		if h.State() != StateActive {
			continue
		}
		cr := packetmodel.CoupleRecord{
			PairCharacterId:   h.PartnerCharacterId(),
			PairCharacterName: h.PartnerName(),
			OwnSN:             h.CashId(),
			PairSN:            h.PartnerCashId(),
		}
		switch h.Type() {
		case TypeCouple:
			rr.Couple = append(rr.Couple, cr)
		case TypeFriendship:
			rr.Friend = append(rr.Friend, packetmodel.FriendRecord{
				CoupleRecord: cr,
				FriendItemId: h.ItemTemplateId(),
			})
		}
	}
	return rr
}

// selectPair applies GetRingSet's selection rule (see its doc comment) for
// one ring type.
func selectPair(halves []Model, t Type, eq equipment.Model) *packetmodel.PairRing {
	var best Model
	var bestPosition slot2.Position
	found := false

	for _, s := range eq.Slots() {
		if !ringSlotPositions[s.Position] || s.CashEquipable == nil {
			continue
		}
		equippedCashId := s.CashEquipable.CashId()
		for _, h := range halves {
			if h.Type() != t || h.State() != StateActive || h.CashId() != equippedCashId {
				continue
			}
			if !found || s.Position > bestPosition || (s.Position == bestPosition && h.CashId() < best.CashId()) {
				best, bestPosition, found = h, s.Position, true
			}
		}
	}

	if !found {
		return nil
	}
	return &packetmodel.PairRing{
		OwnSN:     best.CashId(),
		PartnerSN: best.PartnerCashId(),
		ItemId:    best.ItemTemplateId(),
	}
}
