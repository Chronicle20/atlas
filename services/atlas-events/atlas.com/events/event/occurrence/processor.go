package occurrence

import (
	"atlas-events/event/definition"
	"atlas-events/event/registry"
	"atlas-events/event/transition"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// Processor is the generic entry point onto occurrence persistence. Every
// state or stage change is exposed only as a paired occurrence+transition
// write (FR-O6/FR-T2); there is no method that can write one without the
// other.
type Processor interface {
	CreateFromSeed(d definition.Model, s registry.Seed, triggerRef string) (Model, error)
	ApplyProgress(o Model, p registry.Progress, triggerType, triggerRef string) (Model, error)
	Complete(id uuid.UUID, reason string, triggerType, triggerRef string) (bool, error)
	GetById(id uuid.UUID) (Model, error)
	GetActiveByType(theType string) ([]Model, error)
	GetActiveByVoyage(voyageId uuid.UUID, worldId world.Id, channelId channel.Id) ([]Model, error)
	VisualsInMap(worldId world.Id, channelId channel.Id, mapId _map.Id) ([]Model, error)
	// ListPaged backs GET /events/occurrences (FR-API6): a WHERE-scoped,
	// SQL-paged listing over whichever ListFilters fields are non-zero.
	ListPaged(page model.Page, f ListFilters) (model.Paged[Model], error)
	ObserveMonsterSpawned(occurrenceId uuid.UUID, uniqueId uint32, monsterId uint32) error
	ObserveMonsterGone(occurrenceId uuid.UUID, uniqueId uint32, monsterId uint32) error
	MonsterTally(occurrenceId uuid.UUID) (total int, alive int, err error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
	db  *gorm.DB
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	t := tenant.MustFromContext(ctx)

	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   t,
		db:  db,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

// transitionStage names what a transition row's ToStage should record when
// the domain stage is empty. Some event types have no staged component
// (registry.Seed.Stage may be "") but the transition history still requires
// a non-empty ToStage, so it falls back to the occurrence state.
func transitionStage(stage string, state string) string {
	if stage != "" {
		return stage
	}
	return state
}

// CreateFromSeed inserts a new occurrence, its map scope rows and the
// OCCURRENCE_CREATED transition in one transaction (FR-O6). A concurrency key
// already claimed by an active occurrence returns ErrConcurrencyKeyTaken.
func (p *ProcessorImpl) CreateFromSeed(d definition.Model, s registry.Seed, triggerRef string) (Model, error) {
	m, err := NewBuilder(d.Id(), d.Type()).
		SetState(StateActive).
		SetStage(s.Stage).
		SetContext(s.Context).
		SetWorldId(s.WorldId).
		SetChannelId(s.ChannelId).
		SetVoyageId(s.VoyageId).
		SetConcurrencyKey(s.ConcurrencyKey).
		SetStartedAt(time.Now()).
		SetNextTransitionAt(s.NextTransitionAt).
		Build()
	if err != nil {
		return Model{}, err
	}

	entity, err := ToEntity(m, p.t.Id())
	if err != nil {
		return Model{}, err
	}

	maps := make([]MapEntity, 0, len(s.Maps))
	for _, ms := range s.Maps {
		maps = append(maps, MapEntity{OccurrenceID: entity.ID, MapID: uint32(ms.MapId), Visual: ms.Visual})
	}

	tm, err := transition.NewBuilder(entity.ID, transitionStage(s.Stage, StateActive)).
		SetTrigger(transition.TriggerTypeOccurrenceCreated, triggerRef).
		Build()
	if err != nil {
		return Model{}, err
	}
	transEntity, err := transition.ToEntity(tm, p.t.Id())
	if err != nil {
		return Model{}, err
	}

	result, err := createFromSeed(p.db.WithContext(p.ctx))(entity, maps, transEntity)
	if err != nil {
		return Model{}, err
	}
	return Make(result)
}

// ApplyProgress settles an occurrence into p (the result of a handler's
// Start/Advance) and writes the paired transition row in one transaction
// (FR-O6/FR-T2). p.Terminal completes the occurrence in the same write.
func (p *ProcessorImpl) ApplyProgress(o Model, prog registry.Progress, triggerType, triggerRef string) (Model, error) {
	b := o.Builder().
		SetStage(prog.Stage).
		SetNextTransitionAt(prog.NextTransitionAt)
	if prog.Terminal {
		now := time.Now()
		b = b.SetState(StateCompleted).SetCompletedAt(&now).SetCompletionReason(prog.CompletionReason)
	}
	m, err := b.Build()
	if err != nil {
		return Model{}, err
	}

	entity, err := ToEntity(m, p.t.Id())
	if err != nil {
		return Model{}, err
	}

	tm, err := transition.NewBuilder(entity.ID, transitionStage(prog.Stage, m.State())).
		SetFromStage(o.Stage()).
		SetTrigger(triggerType, triggerRef).
		Build()
	if err != nil {
		return Model{}, err
	}
	transEntity, err := transition.ToEntity(tm, p.t.Id())
	if err != nil {
		return Model{}, err
	}

	result, err := applyProgress(p.db.WithContext(p.ctx))(entity, transEntity)
	if err != nil {
		return Model{}, err
	}
	return Make(result)
}

// Complete is the guarded completion (FR-B20): the first return reports
// whether THIS call won the race. A losing caller gets (false, nil) and must
// skip its cleanup — the winner already ran it.
func (p *ProcessorImpl) Complete(id uuid.UUID, reason string, triggerType, triggerRef string) (bool, error) {
	current, err := p.GetById(id)
	if err != nil {
		return false, err
	}

	at := time.Now()
	tm, err := transition.NewBuilder(id, transitionStage("", StateCompleted)).
		SetFromStage(current.Stage()).
		SetTrigger(triggerType, triggerRef).
		Build()
	if err != nil {
		return false, err
	}
	transEntity, err := transition.ToEntity(tm, p.t.Id())
	if err != nil {
		return false, err
	}

	return complete(p.db.WithContext(p.ctx))(id, reason, at, transEntity)
}

func (p *ProcessorImpl) GetById(id uuid.UUID) (Model, error) {
	return model.Map(Make)(getByIdProvider(id)(p.db.WithContext(p.ctx)))()
}

func (p *ProcessorImpl) GetActiveByType(theType string) ([]Model, error) {
	return model.SliceMap(Make)(getActiveByTypeProvider(theType)(p.db.WithContext(p.ctx)))(model.ParallelMap())()
}

func (p *ProcessorImpl) GetActiveByVoyage(voyageId uuid.UUID, worldId world.Id, channelId channel.Id) ([]Model, error) {
	return model.SliceMap(Make)(getActiveByVoyageProvider(voyageId, worldId, channelId)(p.db.WithContext(p.ctx)))(model.ParallelMap())()
}

func (p *ProcessorImpl) VisualsInMap(worldId world.Id, channelId channel.Id, mapId _map.Id) ([]Model, error) {
	return model.SliceMap(Make)(visualsInMapProvider(worldId, channelId, mapId)(p.db.WithContext(p.ctx)))(model.ParallelMap())()
}

func (p *ProcessorImpl) ListPaged(page model.Page, f ListFilters) (model.Paged[Model], error) {
	ep := listPagedProvider(page, f)(p.db.WithContext(p.ctx))
	return model.MapPaged(Make)(ep)(model.ParallelMap())()
}

// ObserveMonsterSpawned is INSERT-IF-ABSENT, deliberately not an upsert: a
// KILLED that arrived before its CREATED already wrote a dead row, and the
// late CREATED must not resurrect it. The two events share a topic but have
// no ordering guarantee across partitions, so this is a real case (design
// §9.5).
func (p *ProcessorImpl) ObserveMonsterSpawned(occurrenceId uuid.UUID, uniqueId uint32, monsterId uint32) error {
	return p.db.WithContext(p.ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "occurrence_id"}, {Name: "unique_id"}},
		DoNothing: true,
	}).
		Create(&MonsterEntity{
			OccurrenceID: occurrenceId, UniqueID: uniqueId, MonsterID: monsterId,
			Alive: true, ObservedAt: time.Now(),
		}).Error
}

// ObserveMonsterGone is an UPSERT to alive=false: idempotent by construction,
// and correct whether or not CREATED was seen first.
func (p *ProcessorImpl) ObserveMonsterGone(occurrenceId uuid.UUID, uniqueId uint32, monsterId uint32) error {
	return p.db.WithContext(p.ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "occurrence_id"}, {Name: "unique_id"}},
		DoUpdates: clause.Assignments(map[string]any{"alive": false, "observed_at": time.Now()}),
	}).Create(&MonsterEntity{
		OccurrenceID: occurrenceId, UniqueID: uniqueId, MonsterID: monsterId,
		Alive: false, ObservedAt: time.Now(),
	}).Error
}

// MonsterTally reports the current SET-derived counts: total observed, and
// how many remain alive (design §9.5).
func (p *ProcessorImpl) MonsterTally(occurrenceId uuid.UUID) (int, int, error) {
	db := p.db.WithContext(p.ctx)

	var total int64
	if err := db.Model(&MonsterEntity{}).Where("occurrence_id = ?", occurrenceId).Count(&total).Error; err != nil {
		return 0, 0, err
	}

	var alive int64
	if err := db.Model(&MonsterEntity{}).Where("occurrence_id = ? AND alive = ?", occurrenceId, true).Count(&alive).Error; err != nil {
		return 0, 0, err
	}

	return int(total), int(alive), nil
}
