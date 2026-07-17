package reactor

import (
	"atlas-reactors/reactor/data"
	"atlas-reactors/reactor/data/state"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Processor interface {
	GetById(id uint32) (Model, error)
	GetInField(f field.Model) ([]Model, error)
	Create(b *ModelBuilder) error
	DestroyInField(f field.Model)
	Teardown() func()
	DestroyAll() error
	DestroyInTenant(t tenant.Model) model.Operator[[]Model]
	Destroy() model.Operator[Model]
	Hit(reactorId uint32, characterId uint32, skillId uint32) error
	Trigger(r Model, characterId uint32)
	TriggerAndDestroy(r Model, characterId uint32) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetById(id uint32) (Model, error) {
	t := tenant.MustFromContext(p.ctx)
	return GetRegistry().Get(t, id)
}

func (p *ProcessorImpl) GetInField(f field.Model) ([]Model, error) {
	t := tenant.MustFromContext(p.ctx)
	return GetRegistry().GetInField(t, f), nil
}

func (p *ProcessorImpl) Create(b *ModelBuilder) error {
	t := tenant.MustFromContext(p.ctx)
	f := field.NewBuilder(b.worldId, b.channelId, b.mapId).SetInstance(b.instance).Build()
	mk := NewMapKey(f)
	if GetRegistry().IsOnCooldown(t, mk, b.classification, b.x, b.y) {
		p.l.Debugf("Ignoring CREATE for reactor [%d] at (%d,%d) - on cooldown.", b.classification, b.x, b.y)
		return nil
	}

	// Reserve the spatial slot before any expensive work. Prevents two
	// concurrent CREATE commands (e.g. racing map-Enter spawns) from
	// producing duplicate reactors stacked at the same position.
	if !GetRegistry().TryClaimSpot(t, mk, b.classification, b.x, b.y) {
		p.l.Debugf("Ignoring CREATE for reactor [%d] at (%d,%d) in map [%d] instance [%s] - spot already claimed.", b.classification, b.x, b.y, b.mapId, b.instance)
		return nil
	}

	d, err := data.NewProcessor(p.l, p.ctx).GetById(b.Classification())
	if err != nil {
		GetRegistry().ReleaseSpot(t, mk, b.classification, b.x, b.y)
		p.l.WithError(err).Errorf("Unable to retrieve reactor [%d] game data.", b.Classification())
		return err
	}
	b.SetData(d)
	if b.Name() == "" && d.Name() != "" {
		b.SetName(d.Name())
	}
	r, err := GetRegistry().Create(t, b)
	if err != nil {
		GetRegistry().ReleaseSpot(t, mk, b.classification, b.x, b.y)
		p.l.WithError(err).Errorf("Failed to create reactor.")
		return err
	}
	GetRegistry().ClearCooldown(t, mk, r.Classification(), r.X(), r.Y())
	p.l.Debugf("Created reactor [%d] of [%d].", r.Id(), r.Classification())
	scheduleStateTimeout(p.l, p.ctx, r)
	return producer.ProviderImpl(p.l)(p.ctx)(EnvEventStatusTopic)(createdStatusEventProvider(r))
}

func (p *ProcessorImpl) DestroyInField(f field.Model) {
	t := tenant.MustFromContext(p.ctx)
	reactors := GetRegistry().GetInField(t, f)
	mk := NewMapKey(f)
	for _, r := range reactors {
		CancelPendingActivation(r.Id())
		cancelStateTimeout(r.Id())
		GetRegistry().Remove(t, r.Id())
		GetRegistry().ReleaseSpot(t, mk, r.Classification(), r.X(), r.Y())
		_ = producer.ProviderImpl(p.l)(p.ctx)(EnvEventStatusTopic)(destroyedStatusEventProvider(r))
	}
	GetRegistry().ClearAllCooldownsForMap(t, mk)
	GetRegistry().ClearAllSpotsForMap(t, mk)
	p.l.Debugf("Destroyed [%d] reactors and cleared cooldowns for map [%d] instance [%s].", len(reactors), f.MapId(), f.Instance())
}

func (p *ProcessorImpl) Teardown() func() {
	return func() {
		CancelAllPendingActivations()
		cancelAllStateTimeouts()

		ctx, span := otel.GetTracerProvider().Tracer("atlas-reactors").Start(context.Background(), "teardown")
		defer span.End()

		err := NewProcessor(p.l, ctx).DestroyAll()
		if err != nil {
			p.l.WithError(err).Errorf("Error destroying all reactors on teardown.")
		}
	}
}

func allByTenantProvider() model.Provider[map[tenant.Model][]Model] {
	return func() (map[tenant.Model][]Model, error) {
		return GetRegistry().GetAll(), nil
	}
}

func (p *ProcessorImpl) DestroyAll() error {
	return model.ForEachMap(allByTenantProvider(), p.DestroyInTenant, model.ParallelExecute())
}

func (p *ProcessorImpl) DestroyInTenant(t tenant.Model) model.Operator[[]Model] {
	return func(models []Model) error {
		tctx := tenant.WithContext(p.ctx, t)
		return model.ForEachSlice(model.FixedProvider(models), NewProcessor(p.l, tctx).Destroy(), model.ParallelExecute())
	}
}

func (p *ProcessorImpl) Destroy() model.Operator[Model] {
	return func(m Model) error {
		CancelPendingActivation(m.Id())
		cancelStateTimeout(m.Id())
		t := tenant.MustFromContext(p.ctx)
		mk := NewMapKey(m.Field())
		GetRegistry().RecordCooldown(t, mk, m.Classification(), m.X(), m.Y(), m.Delay())
		p.l.Debugf("Recorded cooldown for reactor [%d] at (%d,%d) with delay [%d]ms.", m.Classification(), m.X(), m.Y(), m.Delay())
		GetRegistry().Remove(t, m.Id())
		GetRegistry().ReleaseSpot(t, mk, m.Classification(), m.X(), m.Y())
		return producer.ProviderImpl(p.l)(p.ctx)(EnvEventStatusTopic)(destroyedStatusEventProvider(m))
	}
}

func (p *ProcessorImpl) Hit(reactorId uint32, characterId uint32, skillId uint32) error {
	t := tenant.MustFromContext(p.ctx)
	r, err := p.GetById(reactorId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to get reactor [%d] for hit.", reactorId)
		return err
	}

	// A hit interrupts any pending state timer for this reactor.
	cancelStateTimeout(reactorId)

	// Emit HIT command to atlas-reactor-actions for script processing
	isSkill := skillId != 0
	err = producer.ProviderImpl(p.l)(p.ctx)(EnvCommandReactorActionsTopic)(hitActionsCommandProvider(r, characterId, skillId, isSkill))
	if err != nil {
		p.l.WithError(err).Warnf("Failed to emit HIT command to reactor-actions for reactor [%d].", reactorId)
		// Don't fail the hit - continue with state transition
	}

	stateInfo := r.Data().StateInfo()
	stateEvents, ok := stateInfo[r.State()]
	if !ok || len(stateEvents) == 0 {
		p.l.Debugf("No state events for reactor [%d] state [%d]. Triggering and destroying.", reactorId, r.State())
		return p.TriggerAndDestroy(r, characterId)
	}

	var nextState int8 = -1
	var matchedEventType int32 = 0
	for _, event := range stateEvents {
		if len(event.ActiveSkills()) == 0 || containsSkill(event.ActiveSkills(), skillId) {
			nextState = event.NextState()
			matchedEventType = event.Type()
			break
		}
	}

	if nextState == -1 {
		p.l.Debugf("Reactor [%d] reached terminal state. Triggering and destroying.", reactorId)
		return p.TriggerAndDestroy(r, characterId)
	}

	_, hasNextState := stateInfo[nextState]
	if !hasNextState {
		if persistsAtEndState(matchedEventType) {
			updated, err := GetRegistry().Update(t, reactorId, func(b *ModelBuilder) {
				b.SetState(nextState)
			})
			if err != nil {
				p.l.WithError(err).Errorf("Unable to update reactor [%d] state.", reactorId)
				return err
			}
			p.l.Debugf("Reactor [%d] hit. State changed from [%d] to final state [%d]. Keeping reactor alive (event type %d).", reactorId, r.State(), nextState, matchedEventType)
			// Arm the timer before triggering action emission; local state progression must not be gated on Kafka latency.
			scheduleStateTimeout(p.l, p.ctx, updated)
			p.Trigger(updated, characterId)
			return producer.ProviderImpl(p.l)(p.ctx)(EnvEventStatusTopic)(hitStatusEventProvider(updated, false))
		}
		p.l.Debugf("Reactor [%d] next state [%d] not in state info. Triggering and destroying.", reactorId, nextState)
		return p.TriggerAndDestroy(r, characterId)
	}

	updated, err := GetRegistry().Update(t, reactorId, func(b *ModelBuilder) {
		b.SetState(nextState)
	})
	if err != nil {
		p.l.WithError(err).Errorf("Unable to update reactor [%d] state.", reactorId)
		return err
	}

	// Check if the new state is terminal (all its events lead to non-existent states)
	if isTerminalState(stateInfo, nextState) {
		if persistsAtEndState(matchedEventType) {
			p.l.Debugf("Reactor [%d] hit. State changed from [%d] to terminal state [%d]. Keeping reactor alive (event type %d).", reactorId, r.State(), nextState, matchedEventType)
			scheduleStateTimeout(p.l, p.ctx, updated)
			p.Trigger(updated, characterId)
			return producer.ProviderImpl(p.l)(p.ctx)(EnvEventStatusTopic)(hitStatusEventProvider(updated, false))
		}
		p.l.Debugf("Reactor [%d] hit. State changed from [%d] to terminal state [%d]. Triggering and destroying.", reactorId, r.State(), nextState)
		return p.TriggerAndDestroy(updated, characterId)
	}

	p.l.Debugf("Reactor [%d] hit. State changed from [%d] to [%d].", reactorId, r.State(), nextState)
	scheduleStateTimeout(p.l, p.ctx, updated)
	return producer.ProviderImpl(p.l)(p.ctx)(EnvEventStatusTopic)(hitStatusEventProvider(updated, false))
}

// Trigger emits a TRIGGER command to atlas-reactor-actions without destroying the reactor
func (p *ProcessorImpl) Trigger(r Model, characterId uint32) {
	err := producer.ProviderImpl(p.l)(p.ctx)(EnvCommandReactorActionsTopic)(triggerActionsCommandProvider(r, characterId))
	if err != nil {
		p.l.WithError(err).Warnf("Failed to emit TRIGGER command to reactor-actions for reactor [%d].", r.Id())
	}
}

// TriggerAndDestroy emits a TRIGGER command to atlas-reactor-actions and then destroys the reactor
func (p *ProcessorImpl) TriggerAndDestroy(r Model, characterId uint32) error {
	p.Trigger(r, characterId)
	return p.Destroy()(r)
}

func containsSkill(skills []uint32, skillId uint32) bool {
	for _, s := range skills {
		if s == skillId {
			return true
		}
	}
	return false
}

// persistsAtEndState returns true if a reactor that has just transitioned via
// an event of the given type should remain alive rather than be destroyed.
// Taxonomy (from the wz reactor survey):
//
//	100       item-drop reactors (moonflowers, etc.)
//	101       timer-driven cyclic reactors (Balrog altars, PQ cycles)
//	5, 6, 7   GPQ skill-gated reactors
//
// All other types (0, 1, 2) are breakable hit reactors and destroy on end.
func persistsAtEndState(eventType int32) bool {
	switch eventType {
	case 100, 101, 5, 6, 7:
		return true
	default:
		return false
	}
}

// isTerminalState checks if a state is terminal, meaning all its events
// lead to states that don't exist in the stateInfo map.
func isTerminalState(stateInfo map[int8][]state.Model, s int8) bool {
	events, ok := stateInfo[s]
	if !ok || len(events) == 0 {
		return true
	}
	for _, event := range events {
		if _, exists := stateInfo[event.NextState()]; exists {
			return false
		}
	}
	return true
}
