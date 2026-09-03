package reactor

import (
	"atlas-reactors/character"
	"atlas-reactors/reactor/data"
	"atlas-reactors/reactor/data/state"
	"context"
	"math/rand/v2"

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
	Create(b *Builder) error
	DestroyInField(f field.Model)
	// Teardown builds the shutdown-time destroy-everything sweep. envContext
	// must originate this pod's own environment identity (env.Self()) onto
	// each tenant's context before the per-reactor destroy emits a real
	// Kafka event -- reactor is outside env-domain-guard's permitted
	// atlas-env import list, so main.go threads this in as a plain function
	// value rather than the package importing atlas-env itself.
	Teardown(envContext func(context.Context) context.Context) func()
	DestroyAll(envContext func(context.Context) context.Context) error
	DestroyInTenant(envContext func(context.Context) context.Context, t tenant.Model) model.Operator[[]Model]
	Destroy() model.Operator[Model]
	Hit(reactorId uint32, characterId uint32, skillId uint32) error
	Touch(reactorId uint32, characterId uint32, touching bool) error
	Trigger(r Model, characterId uint32)
	TriggerAndDestroy(r Model, characterId uint32) error
	ResetInField(f field.Model, minState *int8) (int, error)
	ShuffleInField(f field.Model) error
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

// characterProcessor is the seam tests use to stand in for the atlas-character
// REST read. Production resolves the real client.
var characterProcessor = func(l logrus.FieldLogger, ctx context.Context) character.Processor {
	return character.NewProcessor(l, ctx)
}

func (p *ProcessorImpl) GetById(id uint32) (Model, error) {
	t := tenant.MustFromContext(p.ctx)
	return GetRegistry().Get(t, id)
}

func (p *ProcessorImpl) GetInField(f field.Model) ([]Model, error) {
	t := tenant.MustFromContext(p.ctx)
	return GetRegistry().GetInField(t, f), nil
}

func (p *ProcessorImpl) Create(b *Builder) error {
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

func (p *ProcessorImpl) Teardown(envContext func(context.Context) context.Context) func() {
	return func() {
		CancelAllPendingActivations()
		cancelAllStateTimeouts()

		ctx, span := otel.GetTracerProvider().Tracer("atlas-reactors").Start(context.Background(), "teardown")
		defer span.End()

		err := NewProcessor(p.l, ctx).DestroyAll(envContext)
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

func (p *ProcessorImpl) DestroyAll(envContext func(context.Context) context.Context) error {
	return model.ForEachMap(allByTenantProvider(), func(t tenant.Model) model.Operator[[]Model] {
		return p.DestroyInTenant(envContext, t)
	}, model.ParallelExecute())
}

func (p *ProcessorImpl) DestroyInTenant(envContext func(context.Context) context.Context, t tenant.Model) model.Operator[[]Model] {
	return func(models []Model) error {
		tctx := envContext(tenant.WithContext(p.ctx, t))
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

	nextState, matchedEventType := selectNextState(stateEvents, skillId)

	if nextState == -1 {
		p.l.Debugf("Reactor [%d] reached terminal state. Triggering and destroying.", reactorId)
		return p.TriggerAndDestroy(r, characterId)
	}

	return p.advance(r, characterId, nextState, matchedEventType)
}

// Touch handles a TOUCHING_REACTOR command. On leave (touching == false), it
// releases the character's touch latch. On enter, it runs the rejection
// ladder (design §6.1) before advancing state.
func (p *ProcessorImpl) Touch(reactorId uint32, characterId uint32, touching bool) error {
	t := tenant.MustFromContext(p.ctx)
	if !touching {
		GetRegistry().ClearTouch(t, reactorId, characterId)
		p.l.Debugf("Character [%d] left touch area of reactor [%d].", characterId, reactorId)
		return nil
	}

	r, err := p.GetById(reactorId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to get reactor [%d] for touch.", reactorId)
		return err
	}

	if !r.Data().ActivateByTouch() {
		p.l.Debugf("Reactor [%d] is not touch-activated. Ignoring touch from character [%d].", reactorId, characterId)
		return nil
	}

	a, ok := r.Data().TouchArea(r.State())
	if !ok {
		p.l.Debugf("Reactor [%d] has no touch area defined for state [%d]. Ignoring touch from character [%d].", reactorId, r.State(), characterId)
		return nil
	}

	cx, cy, err := characterProcessor(p.l, p.ctx).Position(characterId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to get position of character [%d] for touch of reactor [%d].", characterId, reactorId)
		return nil
	}

	if cx < r.X()+a.TL().X() || cx > r.X()+a.BR().X() ||
		cy < r.Y()+a.TL().Y() || cy > r.Y()+a.BR().Y() {
		p.l.Debugf("Character [%d] at (%d,%d) is outside touch area of reactor [%d]. Ignoring.", characterId, cx, cy, reactorId)
		return nil
	}

	if !GetRegistry().TryLatchTouch(t, reactorId, characterId) {
		p.l.Debugf("Character [%d] already latched to reactor [%d]. Ignoring duplicate touch.", characterId, reactorId)
		return nil
	}

	cancelStateTimeout(reactorId)

	err = producer.ProviderImpl(p.l)(p.ctx)(EnvCommandReactorActionsTopic)(touchActionsCommandProvider(r, characterId))
	if err != nil {
		p.l.WithError(err).Warnf("Failed to emit TOUCH command to reactor-actions for reactor [%d].", reactorId)
	}

	stateEvents, ok := r.Data().StateInfo()[r.State()]
	if !ok || len(stateEvents) == 0 {
		p.l.Debugf("No state events for reactor [%d] state [%d] on touch. No-op.", reactorId, r.State())
		return nil
	}
	return p.advance(r, characterId, stateEvents[0].NextState(), stateEvents[0].Type())
}

// selectNextState applies the hit path's skill-gating predicate to a state's
// events. Returns (-1, 0) when no event matches. Touch does NOT use this --
// see Touch's own selection (FR-16).
func selectNextState(stateEvents []state.Model, skillId uint32) (int8, int32) {
	for _, event := range stateEvents {
		if len(event.ActiveSkills()) == 0 || containsSkill(event.ActiveSkills(), skillId) {
			return event.NextState(), event.Type()
		}
	}
	return -1, 0
}

func (p *ProcessorImpl) advance(r Model, characterId uint32, nextState int8, matchedEventType int32) error {
	t := tenant.MustFromContext(p.ctx)
	reactorId := r.Id()
	stateInfo := r.Data().StateInfo()

	_, hasNextState := stateInfo[nextState]
	if !hasNextState {
		if persistsAtEndState(matchedEventType) {
			updated, err := GetRegistry().Update(t, reactorId, func(b *Builder) {
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

	updated, err := GetRegistry().Update(t, reactorId, func(b *Builder) {
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

// ResetInField resets every matching reactor on the field to state 0,
// mirroring Cosmic's MapleMap.resetReactors(List<Reactor>) (MapleMap.java:1563).
// Cosmic skips any reactor whose forceDelayedRespawn() returns true --
// atlas-reactors has no analogue of that flag, so every reactor in the
// field is a candidate. When minState is non-nil, only reactors whose
// State() is at least *minState are reset; this is 926120300.js's
// getInactiveReactors filter (state >= 7) computed in script and passed to
// the single resetReactors(List) overload -- there is no state-filtered
// Java overload, so this is modelled as one reset with an optional
// minimum-state filter rather than two methods. Returns the count reset.
func (p *ProcessorImpl) ResetInField(f field.Model, minState *int8) (int, error) {
	t := tenant.MustFromContext(p.ctx)
	reactors := GetRegistry().GetInField(t, f)
	count := 0
	for _, r := range reactors {
		if minState != nil && r.State() < *minState {
			continue
		}
		cancelStateTimeout(r.Id())
		updated, err := GetRegistry().Update(t, r.Id(), func(b *Builder) {
			b.SetState(0)
		})
		if err != nil {
			p.l.WithError(err).Errorf("Unable to reset reactor [%d].", r.Id())
			return count, err
		}
		count++
		if err := producer.ProviderImpl(p.l)(p.ctx)(EnvEventStatusTopic)(hitStatusEventProvider(updated, false)); err != nil {
			p.l.WithError(err).Warnf("Failed to emit status event for reset reactor [%d].", r.Id())
		}
	}
	p.l.Debugf("Reset [%d] reactors in map [%d] instance [%s].", count, f.MapId(), f.Instance())
	return count, nil
}

// ShuffleInField randomly permutes the positions of every reactor on the
// field, mirroring Cosmic's MapleMap.shuffleReactors() (MapleMap.java:1580).
// Only (x,y) is reassigned onto the same reactor objects -- ids, states and
// identities are untouched.
func (p *ProcessorImpl) ShuffleInField(f field.Model) error {
	t := tenant.MustFromContext(p.ctx)
	reactors := GetRegistry().GetInField(t, f)
	if len(reactors) < 2 {
		return nil
	}

	type position struct {
		x int16
		y int16
	}
	positions := make([]position, len(reactors))
	for i, r := range reactors {
		positions[i] = position{x: r.X(), y: r.Y()}
	}
	rand.Shuffle(len(positions), func(i, j int) {
		positions[i], positions[j] = positions[j], positions[i]
	})

	for i, r := range reactors {
		pos := positions[i]
		if _, err := GetRegistry().Update(t, r.Id(), func(b *Builder) {
			b.SetPosition(pos.x, pos.y)
		}); err != nil {
			p.l.WithError(err).Errorf("Unable to shuffle reactor [%d] position.", r.Id())
			return err
		}
	}
	p.l.Debugf("Shuffled positions of [%d] reactors in map [%d] instance [%s].", len(reactors), f.MapId(), f.Instance())
	return nil
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
