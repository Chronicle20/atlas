package trade

import (
	"context"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Processor owns every trade-room operation. REST handlers go through it rather
// than reaching into the registry directly (DOM-14). Only the reads exist at
// this point; the Kafka-driven commands join the same interface later.
type Processor interface {
	// RoomsForTenant returns every live room the request's tenant owns.
	RoomsForTenant() []Room

	// RoomById returns the room with the given id, scoped to the request's
	// tenant. A settled or cancelled room is gone, so a miss is reported rather
	// than a stale snapshot.
	RoomById(id uuid.UUID) (Room, bool)

	// RoomForCharacter returns the room the character occupies as either side.
	RoomForCharacter(characterId character.Id) (Room, bool)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
	reg *Registry
}

// NewProcessor resolves the tenant from ctx once; every registry read the
// processor issues is partitioned by that tenant.
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx, t: tenant.MustFromContext(ctx), reg: GetRegistry()}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) RoomsForTenant() []Room { return p.reg.All(p.t) }

func (p *ProcessorImpl) RoomById(id uuid.UUID) (Room, bool) { return p.reg.Get(p.t, id) }

func (p *ProcessorImpl) RoomForCharacter(characterId character.Id) (Room, bool) {
	return p.reg.GetByMember(p.t, characterId)
}
