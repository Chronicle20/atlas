package monster

import (
	"atlas-monsters/kafka/producer"
	_map "atlas-monsters/map"
	"atlas-monsters/monster/information"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
)

// Processor defines the interface for monster processing operations
type Processor interface {
	// Providers
	ByIdProvider(monsterId uint32) model.Provider[Model]
	ByMapProvider(worldId byte, channelId byte, mapId uint32) model.Provider[[]Model]
	ControlledInMapProvider(worldId byte, channelId byte, mapId uint32) model.Provider[[]Model]
	NotControlledInMapProvider(worldId byte, channelId byte, mapId uint32) model.Provider[[]Model]
	ControlledByCharacterInMapProvider(worldId byte, channelId byte, mapId uint32, characterId uint32) model.Provider[[]Model]

	// Queries
	GetById(monsterId uint32) (Model, error)
	GetInMap(worldId byte, channelId byte, mapId uint32) ([]Model, error)

	// Commands
	Create(worldId byte, channelId byte, mapId uint32, input RestModel) (Model, error)
	StartControl(uniqueId uint32, controllerId uint32) (Model, error)
	StopControl(m Model) error
	FindNextController(idp model.Provider[[]uint32]) model.Operator[Model]
	Damage(id uint32, characterId uint32, damage uint32)
	Move(id uint32, x int16, y int16, stance byte) error
	Destroy(uniqueId uint32) error
	DestroyInMap(worldId byte, channelId byte, mapId uint32) error
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
}

// NewProcessor creates a new Processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
	}
}

// ByIdProvider returns a provider for a monster by ID
func (p *ProcessorImpl) ByIdProvider(monsterId uint32) model.Provider[Model] {
	return func() (Model, error) {
		return GetMonsterRegistry().GetMonster(p.t, monsterId)
	}
}

// ByMapProvider returns a provider for monsters in a map
func (p *ProcessorImpl) ByMapProvider(worldId byte, channelId byte, mapId uint32) model.Provider[[]Model] {
	return func() ([]Model, error) {
		return GetMonsterRegistry().GetMonstersInMap(p.t, worldId, channelId, mapId), nil
	}
}

// ControlledInMapProvider returns a provider for controlled monsters in a map
func (p *ProcessorImpl) ControlledInMapProvider(worldId byte, channelId byte, mapId uint32) model.Provider[[]Model] {
	return model.FilteredProvider(p.ByMapProvider(worldId, channelId, mapId), model.Filters(Controlled))
}

// NotControlledInMapProvider returns a provider for uncontrolled monsters in a map
func (p *ProcessorImpl) NotControlledInMapProvider(worldId byte, channelId byte, mapId uint32) model.Provider[[]Model] {
	return model.FilteredProvider(p.ByMapProvider(worldId, channelId, mapId), model.Filters(NotControlled))
}

// ControlledByCharacterInMapProvider returns a provider for monsters controlled by a specific character
func (p *ProcessorImpl) ControlledByCharacterInMapProvider(worldId byte, channelId byte, mapId uint32, characterId uint32) model.Provider[[]Model] {
	return model.FilteredProvider(p.ByMapProvider(worldId, channelId, mapId), model.Filters(IsControlledBy(characterId)))
}

// GetById gets a monster by ID
func (p *ProcessorImpl) GetById(monsterId uint32) (Model, error) {
	return p.ByIdProvider(monsterId)()
}

// GetInMap gets all monsters in a map
func (p *ProcessorImpl) GetInMap(worldId byte, channelId byte, mapId uint32) ([]Model, error) {
	return p.ByMapProvider(worldId, channelId, mapId)()
}

// Create creates a new monster in a map
func (p *ProcessorImpl) Create(worldId byte, channelId byte, mapId uint32, input RestModel) (Model, error) {
	p.l.Debugf("Attempting to create monster [%d] in world [%d] channel [%d] map [%d].", input.MonsterId, worldId, channelId, mapId)
	ma, err := information.GetById(p.l)(p.ctx)(input.MonsterId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to retrieve information necessary to create monster [%d].", input.MonsterId)
		return Model{}, err
	}

	m := GetMonsterRegistry().CreateMonster(p.t, worldId, channelId, mapId, input.MonsterId, input.X, input.Y, input.Fh, 5, input.Team, ma.HP(), ma.MP())

	cid, err := p.getControllerCandidate(worldId, channelId, mapId, _map.CharacterIdsInMapProvider(p.l)(p.ctx)(worldId, channelId, mapId))
	if err == nil {
		p.l.Debugf("Created monster [%d] with id [%d] will be controlled by [%d].", m.MonsterId(), m.UniqueId(), cid)
		m, err = p.StartControl(m.UniqueId(), cid)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to start [%d] controlling [%d] in world [%d] channel [%d] map [%d].", cid, m.UniqueId(), m.WorldId(), m.ChannelId(), m.MapId())
		}
	}

	p.l.Debugf("Created monster [%d] in world [%d] channel [%d] map [%d]. Emitting Monster Status.", input.MonsterId, worldId, channelId, mapId)
	_ = producer.ProviderImpl(p.l)(p.ctx)(EnvEventTopicMonsterStatus)(createdStatusEventProvider(m.WorldId(), m.ChannelId(), m.MapId(), m.UniqueId(), m.MonsterId()))
	return m, nil
}

// getControllerCandidate finds the best character to control monsters in a map
func (p *ProcessorImpl) getControllerCandidate(worldId byte, channelId byte, mapId uint32, idp model.Provider[[]uint32]) (uint32, error) {
	p.l.Debugf("Identifying controller candidate for monsters in world [%d] channel [%d] map [%d].", worldId, channelId, mapId)

	controlCounts, err := model.CollectToMap(idp, characterIdKey, zeroValue)()
	if err != nil {
		p.l.WithError(err).Errorf("Unable to initialize controller candidate map.")
		return 0, err
	}
	err = model.ForEachSlice(p.ControlledInMapProvider(worldId, channelId, mapId), func(m Model) error {
		controlCounts[m.ControlCharacterId()] += 1
		return nil
	})

	var index = uint32(0)
	for key, val := range controlCounts {
		if index == 0 {
			index = key
		} else if val < controlCounts[index] {
			index = key
		}
	}

	if index == 0 {
		return 0, errors.New("should not get here")
	}
	p.l.Debugf("Controller candidate has been determined. Character [%d].", index)
	return index, nil
}

// FindNextController returns an operator that finds and assigns the next controller for a monster
func (p *ProcessorImpl) FindNextController(idp model.Provider[[]uint32]) model.Operator[Model] {
	return func(m Model) error {
		cid, err := p.getControllerCandidate(m.WorldId(), m.ChannelId(), m.MapId(), idp)
		if err != nil {
			return err
		}

		_, err = p.StartControl(m.UniqueId(), cid)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to start [%d] controlling [%d] in world [%d] channel [%d] map [%d].", cid, m.UniqueId(), m.WorldId(), m.ChannelId(), m.MapId())
		}
		return err
	}
}

// StartControl starts a character controlling a monster
func (p *ProcessorImpl) StartControl(uniqueId uint32, controllerId uint32) (Model, error) {
	m, err := p.GetById(uniqueId)
	if err != nil {
		return Model{}, err
	}

	if m.ControlCharacterId() != 0 {
		err = p.StopControl(m)
		if err != nil {
			return Model{}, err
		}
	}

	m, err = p.GetById(uniqueId)
	if err != nil {
		return Model{}, err
	}

	m, err = GetMonsterRegistry().ControlMonster(p.t, m.UniqueId(), controllerId)
	if err == nil {
		_ = producer.ProviderImpl(p.l)(p.ctx)(EnvEventTopicMonsterStatus)(startControlStatusEventProvider(m))
	}
	return m, err
}

// StopControl stops a character from controlling a monster
func (p *ProcessorImpl) StopControl(m Model) error {
	m, err := GetMonsterRegistry().ClearControl(p.t, m.UniqueId())
	if err == nil {
		_ = producer.ProviderImpl(p.l)(p.ctx)(EnvEventTopicMonsterStatus)(stopControlStatusEventProvider(m.WorldId(), m.ChannelId(), m.MapId(), m.UniqueId(), m.MonsterId(), m.ControlCharacterId()))
	}
	return err
}

// Damage applies damage to a monster
func (p *ProcessorImpl) Damage(id uint32, characterId uint32, damage uint32) {
	m, err := GetMonsterRegistry().GetMonster(p.t, id)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to get monster [%d].", id)
		return
	}
	if !m.Alive() {
		p.l.Debugf("Character [%d] trying to apply damage to an already dead monster [%d].", characterId, id)
		return
	}

	s, err := GetMonsterRegistry().ApplyDamage(p.t, characterId, damage, m.UniqueId())
	if err != nil {
		p.l.WithError(err).Errorf("Error applying damage to monster %d from character %d.", m.UniqueId(), characterId)
		return
	}

	if s.Killed {
		err = producer.ProviderImpl(p.l)(p.ctx)(EnvEventTopicMonsterStatus)(killedStatusEventProvider(s.Monster.WorldId(), s.Monster.ChannelId(), s.Monster.MapId(), s.Monster.UniqueId(), s.Monster.MonsterId(), s.Monster.X(), s.Monster.Y(), s.CharacterId, s.Monster.DamageSummary()))
		if err != nil {
			p.l.WithError(err).Errorf("Monster [%d] killed, but unable to display that for the characters in the map.", s.Monster.UniqueId())
		}
		_, err = GetMonsterRegistry().RemoveMonster(p.t, s.Monster.UniqueId())
		if err != nil {
			p.l.WithError(err).Errorf("Monster [%d] killed, but not removed from registry.", s.Monster.UniqueId())
		}
		return
	}

	if characterId != s.Monster.ControlCharacterId() {
		dl := s.Monster.DamageLeader() == characterId
		p.l.Debugf("Character [%d] has become damage leader. They should now control the monster.", characterId)
		if dl {
			m, err := p.GetById(s.Monster.UniqueId())
			if err != nil {
				return
			}

			err = p.StopControl(m)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to stop [%d] from controlling monster [%d].", s.Monster.ControlCharacterId(), s.Monster.UniqueId())
			}
			_, err = p.StartControl(m.UniqueId(), characterId)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to start [%d] controlling monster [%d].", characterId, m.UniqueId())
			}
		}
	}

	err = producer.ProviderImpl(p.l)(p.ctx)(EnvEventTopicMonsterStatus)(damagedStatusEventProvider(s.Monster.WorldId(), s.Monster.ChannelId(), s.Monster.MapId(), s.Monster.UniqueId(), s.Monster.MonsterId(), s.Monster.X(), s.Monster.Y(), s.CharacterId, s.Monster.DamageSummary()))
	if err != nil {
		p.l.WithError(err).Errorf("Monster [%d] damaged, but unable to display that for the characters in the map.", s.Monster.UniqueId())
	}
}

// Move moves a monster to a new position
func (p *ProcessorImpl) Move(id uint32, x int16, y int16, stance byte) error {
	GetMonsterRegistry().MoveMonster(p.t, id, x, y, stance)
	return nil
}

// Destroy destroys a monster
func (p *ProcessorImpl) Destroy(uniqueId uint32) error {
	m, err := GetMonsterRegistry().RemoveMonster(p.t, uniqueId)
	if err != nil {
		return err
	}

	return producer.ProviderImpl(p.l)(p.ctx)(EnvEventTopicMonsterStatus)(destroyedStatusEventProvider(m.WorldId(), m.ChannelId(), m.MapId(), m.UniqueId(), m.MonsterId()))
}

// DestroyInMap destroys all monsters in a map
func (p *ProcessorImpl) DestroyInMap(worldId byte, channelId byte, mapId uint32) error {
	return model.ForEachSlice(model.SliceMap[Model, uint32](IdTransformer)(p.ByMapProvider(worldId, channelId, mapId))(model.ParallelMap()), p.Destroy, model.ParallelExecute())
}

// Helper functions

func zeroValue(id uint32) int {
	return 0
}

func characterIdKey(id uint32) uint32 {
	return id
}

func IdTransformer(m Model) (uint32, error) {
	return m.UniqueId(), nil
}

// Filter functions

func Controlled(m Model) bool {
	return m.ControlCharacterId() != 0
}

func NotControlled(m Model) bool {
	return m.ControlCharacterId() == 0
}

func IsControlledBy(id uint32) model.Filter[Model] {
	return func(m Model) bool {
		return m.ControlCharacterId() == id
	}
}

// Lifecycle functions for service shutdown

func allByTenantProvider() model.Provider[map[tenant.Model][]Model] {
	return func() (map[tenant.Model][]Model, error) {
		return GetMonsterRegistry().GetMonsters(), nil
	}
}

func destroyInTenant(l logrus.FieldLogger) func(ctx context.Context) func(t tenant.Model) model.Operator[[]Model] {
	return func(ctx context.Context) func(t tenant.Model) model.Operator[[]Model] {
		return func(t tenant.Model) model.Operator[[]Model] {
			return func(models []Model) error {
				tctx := tenant.WithContext(ctx, t)
				p := NewProcessor(l, tctx)
				idp := model.SliceMap(IdTransformer)(model.FixedProvider(models))(model.ParallelMap())
				return model.ForEachSlice(idp, p.Destroy, model.ParallelExecute())
			}
		}
	}
}

func DestroyAll(l logrus.FieldLogger, ctx context.Context) error {
	return model.ForEachMap(allByTenantProvider(), destroyInTenant(l)(ctx), model.ParallelExecute())
}

func Teardown(l logrus.FieldLogger) func() {
	return func() {
		ctx, span := otel.GetTracerProvider().Tracer("atlas-monsters").Start(context.Background(), "teardown")
		defer span.End()

		err := DestroyAll(l, ctx)
		if err != nil {
			l.WithError(err).Errorf("Error destroying all monsters on teardown.")
		}
	}
}
