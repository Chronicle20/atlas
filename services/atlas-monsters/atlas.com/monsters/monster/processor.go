package monster

import (
	"atlas-monsters/kafka/producer"
	_map "atlas-monsters/map"
	"atlas-monsters/monster/information"
	"context"
	"errors"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
)

// Processor defines the interface for monster processing operations
type Processor interface {
	// Providers
	ByIdProvider(monsterId uint32) model.Provider[Model]
	ByFieldProvider(f field.Model) model.Provider[[]Model]
	ControlledInFieldProvider(f field.Model) model.Provider[[]Model]
	NotControlledInFieldProvider(f field.Model) model.Provider[[]Model]
	ControlledByCharacterInFieldProvider(f field.Model, characterId uint32) model.Provider[[]Model]

	// Queries
	GetById(monsterId uint32) (Model, error)
	GetInField(f field.Model) ([]Model, error)

	// Commands
	Create(f field.Model, input RestModel) (Model, error)
	StartControl(uniqueId uint32, controllerId uint32) (Model, error)
	StopControl(m Model) error
	FindNextController(idp model.Provider[[]uint32]) model.Operator[Model]
	Damage(id uint32, characterId uint32, damage uint32)
	Move(id uint32, x int16, y int16, stance byte) error
	Destroy(uniqueId uint32) error
	DestroyInField(f field.Model) error
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

// ByFieldProvider returns a provider for monsters in a field
func (p *ProcessorImpl) ByFieldProvider(f field.Model) model.Provider[[]Model] {
	return func() ([]Model, error) {
		return GetMonsterRegistry().GetMonstersInMap(p.t, f), nil
	}
}

// ControlledInFieldProvider returns a provider for controlled monsters in a field
func (p *ProcessorImpl) ControlledInFieldProvider(f field.Model) model.Provider[[]Model] {
	return model.FilteredProvider(p.ByFieldProvider(f), model.Filters(Controlled))
}

// NotControlledInFieldProvider returns a provider for uncontrolled monsters in a field
func (p *ProcessorImpl) NotControlledInFieldProvider(f field.Model) model.Provider[[]Model] {
	return model.FilteredProvider(p.ByFieldProvider(f), model.Filters(NotControlled))
}

// ControlledByCharacterInFieldProvider returns a provider for monsters controlled by a specific character
func (p *ProcessorImpl) ControlledByCharacterInFieldProvider(f field.Model, characterId uint32) model.Provider[[]Model] {
	return model.FilteredProvider(p.ByFieldProvider(f), model.Filters(IsControlledBy(characterId)))
}

// GetById gets a monster by ID
func (p *ProcessorImpl) GetById(monsterId uint32) (Model, error) {
	return p.ByIdProvider(monsterId)()
}

// GetInField gets all monsters in a field
func (p *ProcessorImpl) GetInField(f field.Model) ([]Model, error) {
	return p.ByFieldProvider(f)()
}

// Create creates a new monster in a field
func (p *ProcessorImpl) Create(f field.Model, input RestModel) (Model, error) {
	p.l.Debugf("Attempting to create monster [%d] in field [%s].", input.MonsterId, f.Id())
	ma, err := information.GetById(p.l)(p.ctx)(input.MonsterId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to retrieve information necessary to create monster [%d].", input.MonsterId)
		return Model{}, err
	}

	m := GetMonsterRegistry().CreateMonster(p.t, f, input.MonsterId, input.X, input.Y, input.Fh, 5, input.Team, ma.HP(), ma.MP())

	cid, err := p.getControllerCandidate(f, _map.CharacterIdsInFieldProvider(p.l)(p.ctx)(f))
	if err == nil {
		p.l.Debugf("Created monster [%d] with id [%d] will be controlled by [%d].", m.MonsterId(), m.UniqueId(), cid)
		m, err = p.StartControl(m.UniqueId(), cid)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to start [%d] controlling [%d] in field [%s].", cid, m.UniqueId(), m.Field().Id())
		}
	}

	p.l.Debugf("Created monster [%d] in field [%s]. Emitting Monster Status.", input.MonsterId, f.Id())
	_ = producer.ProviderImpl(p.l)(p.ctx)(EnvEventTopicMonsterStatus)(createdStatusEventProvider(m))
	return m, nil
}

// getControllerCandidate finds the best character to control monsters in a field
func (p *ProcessorImpl) getControllerCandidate(f field.Model, idp model.Provider[[]uint32]) (uint32, error) {
	p.l.Debugf("Identifying controller candidate for monsters in field [%s].", f.Id())

	controlCounts, err := model.CollectToMap(idp, characterIdKey, zeroValue)()
	if err != nil {
		p.l.WithError(err).Errorf("Unable to initialize controller candidate map.")
		return 0, err
	}
	err = model.ForEachSlice(p.ControlledInFieldProvider(f), func(m Model) error {
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
		cid, err := p.getControllerCandidate(m.Field(), idp)
		if err != nil {
			return err
		}

		_, err = p.StartControl(m.UniqueId(), cid)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to start [%d] controlling [%d] in field [%s].", cid, m.UniqueId(), m.Field().Id())
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
	oldControllerId := m.ControlCharacterId()
	m, err := GetMonsterRegistry().ClearControl(p.t, m.UniqueId())
	if err == nil {
		_ = producer.ProviderImpl(p.l)(p.ctx)(EnvEventTopicMonsterStatus)(stopControlStatusEventProvider(m, oldControllerId))
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
		err = producer.ProviderImpl(p.l)(p.ctx)(EnvEventTopicMonsterStatus)(killedStatusEventProvider(s.Monster, s.CharacterId, s.Monster.DamageSummary()))
		if err != nil {
			p.l.WithError(err).Errorf("Monster [%d] killed, but unable to display that for the characters in the field.", s.Monster.UniqueId())
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

	err = producer.ProviderImpl(p.l)(p.ctx)(EnvEventTopicMonsterStatus)(damagedStatusEventProvider(s.Monster, s.CharacterId, s.Monster.DamageSummary()))
	if err != nil {
		p.l.WithError(err).Errorf("Monster [%d] damaged, but unable to display that for the characters in the field.", s.Monster.UniqueId())
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

	return producer.ProviderImpl(p.l)(p.ctx)(EnvEventTopicMonsterStatus)(destroyedStatusEventProvider(m))
}

// DestroyInField destroys all monsters in a field
func (p *ProcessorImpl) DestroyInField(f field.Model) error {
	return model.ForEachSlice(model.SliceMap[Model, uint32](IdTransformer)(p.ByFieldProvider(f))(model.ParallelMap()), p.Destroy, model.ParallelExecute())
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
