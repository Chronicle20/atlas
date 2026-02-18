package reactor

import (
	"atlas-reactors/kafka/producer"
	"atlas-reactors/reactor/data"
	"atlas-reactors/reactor/data/state"
	"context"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
)

func GetById(l logrus.FieldLogger) func(ctx context.Context) func(id uint32) (Model, error) {
	return func(ctx context.Context) func(id uint32) (Model, error) {
		return func(id uint32) (Model, error) {
			return GetRegistry().Get(id)
		}
	}
}

func GetInField(l logrus.FieldLogger) func(ctx context.Context) func(f field.Model) ([]Model, error) {
	return func(ctx context.Context) func(f field.Model) ([]Model, error) {
		t := tenant.MustFromContext(ctx)
		return func(f field.Model) ([]Model, error) {
			return GetRegistry().GetInField(t, f), nil
		}
	}
}

func Create(l logrus.FieldLogger) func(ctx context.Context) func(b *ModelBuilder) error {
	return func(ctx context.Context) func(b *ModelBuilder) error {
		t := tenant.MustFromContext(ctx)
		return func(b *ModelBuilder) error {
			f := field.NewBuilder(b.worldId, b.channelId, b.mapId).SetInstance(b.instance).Build()
			mk := NewMapKey(f)
			if GetRegistry().IsOnCooldown(t, mk, b.classification, b.x, b.y) {
				l.Debugf("Ignoring CREATE for reactor [%d] at (%d,%d) - on cooldown.", b.classification, b.x, b.y)
				return nil
			}

			d, err := data.GetById(l)(ctx)(b.Classification())
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve reactor [%d] game data.", b.Classification())
				return err
			}
			b.SetData(d)
			if b.Name() == "" && d.Name() != "" {
				b.SetName(d.Name())
			}
			r, err := GetRegistry().Create(t, b)
			if err != nil {
				l.WithError(err).Errorf("Failed to create reactor.")
				return err
			}
			GetRegistry().ClearCooldown(t, mk, r.Classification(), r.X(), r.Y())
			l.Debugf("Created reactor [%d] of [%d].", r.Id(), r.Classification())
			return producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(createdStatusEventProvider(r))
		}
	}
}

func ClearCooldownsInField(l logrus.FieldLogger) func(ctx context.Context) func(f field.Model) {
	return func(ctx context.Context) func(f field.Model) {
		t := tenant.MustFromContext(ctx)
		return func(f field.Model) {
			mk := NewMapKey(f)
			GetRegistry().ClearAllCooldownsForMap(t, mk)
			l.Debugf("Cleared all reactor cooldowns for map [%d] instance [%s].", f.MapId(), f.Instance())
		}
	}
}

func Teardown(l logrus.FieldLogger) func() {
	return func() {
		CancelAllPendingActivations()

		ctx, span := otel.GetTracerProvider().Tracer("atlas-reactors").Start(context.Background(), "teardown")
		defer span.End()

		err := DestroyAll(l)(ctx)
		if err != nil {
			l.WithError(err).Errorf("Error destroying all reactors on teardown.")
		}
	}
}

func allByTenantProvider() model.Provider[map[tenant.Model][]Model] {
	return func() (map[tenant.Model][]Model, error) {
		return GetRegistry().GetAll(), nil
	}
}

func DestroyAll(l logrus.FieldLogger) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		return model.ForEachMap(allByTenantProvider(), DestroyInTenant(l)(ctx), model.ParallelExecute())
	}
}

func DestroyInTenant(l logrus.FieldLogger) func(ctx context.Context) func(t tenant.Model) model.Operator[[]Model] {
	return func(ctx context.Context) func(t tenant.Model) model.Operator[[]Model] {
		return func(t tenant.Model) model.Operator[[]Model] {
			return func(models []Model) error {
				tctx := tenant.WithContext(ctx, t)
				return model.ForEachSlice(model.FixedProvider(models), Destroy(l)(tctx), model.ParallelExecute())
			}
		}
	}
}

func Destroy(l logrus.FieldLogger) func(ctx context.Context) model.Operator[Model] {
	return func(ctx context.Context) model.Operator[Model] {
		return func(m Model) error {
			CancelPendingActivation(m.Id())
			t := tenant.MustFromContext(ctx)
			mk := NewMapKey(m.Field())
			GetRegistry().RecordCooldown(t, mk, m.Classification(), m.X(), m.Y(), m.Delay())
			l.Debugf("Recorded cooldown for reactor [%d] at (%d,%d) with delay [%d]ms.", m.Classification(), m.X(), m.Y(), m.Delay())
			GetRegistry().Remove(t, m.Id())
			return producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(destroyedStatusEventProvider(m))
		}
	}
}

func Hit(l logrus.FieldLogger) func(ctx context.Context) func(reactorId uint32, characterId uint32, skillId uint32) error {
	return func(ctx context.Context) func(reactorId uint32, characterId uint32, skillId uint32) error {
		return func(reactorId uint32, characterId uint32, skillId uint32) error {
			r, err := GetById(l)(ctx)(reactorId)
			if err != nil {
				l.WithError(err).Errorf("Unable to get reactor [%d] for hit.", reactorId)
				return err
			}

			// Emit HIT command to atlas-reactor-actions for script processing
			isSkill := skillId != 0
			err = producer.ProviderImpl(l)(ctx)(EnvCommandReactorActionsTopic)(hitActionsCommandProvider(r, characterId, skillId, isSkill))
			if err != nil {
				l.WithError(err).Warnf("Failed to emit HIT command to reactor-actions for reactor [%d].", reactorId)
				// Don't fail the hit - continue with state transition
			}

			stateInfo := r.Data().StateInfo()
			stateEvents, ok := stateInfo[r.State()]
			if !ok || len(stateEvents) == 0 {
				l.Debugf("No state events for reactor [%d] state [%d]. Triggering and destroying.", reactorId, r.State())
				return TriggerAndDestroy(l)(ctx)(r, characterId)
			}

			var nextState int8 = -1
			for _, event := range stateEvents {
				if len(event.ActiveSkills()) == 0 || containsSkill(event.ActiveSkills(), skillId) {
					nextState = event.NextState()
					break
				}
			}

			if nextState == -1 {
				l.Debugf("Reactor [%d] reached terminal state. Triggering and destroying.", reactorId)
				return TriggerAndDestroy(l)(ctx)(r, characterId)
			}

			_, hasNextState := stateInfo[nextState]
			if !hasNextState {
				l.Debugf("Reactor [%d] next state [%d] not in state info. Triggering and destroying.", reactorId, nextState)
				return TriggerAndDestroy(l)(ctx)(r, characterId)
			}

			updated, err := GetRegistry().Update(reactorId, func(b *ModelBuilder) {
				b.SetState(nextState)
			})
			if err != nil {
				l.WithError(err).Errorf("Unable to update reactor [%d] state.", reactorId)
				return err
			}

			// Check if the new state is terminal (all its events lead to non-existent states)
			if isTerminalState(stateInfo, nextState) {
				l.Debugf("Reactor [%d] hit. State changed from [%d] to terminal state [%d]. Triggering and destroying.", reactorId, r.State(), nextState)
				return TriggerAndDestroy(l)(ctx)(updated, characterId)
			}

			l.Debugf("Reactor [%d] hit. State changed from [%d] to [%d].", reactorId, r.State(), nextState)
			return producer.ProviderImpl(l)(ctx)(EnvEventStatusTopic)(hitStatusEventProvider(updated, false))
		}
	}
}

// TriggerAndDestroy emits a TRIGGER command to atlas-reactor-actions and then destroys the reactor
func TriggerAndDestroy(l logrus.FieldLogger) func(ctx context.Context) func(r Model, characterId uint32) error {
	return func(ctx context.Context) func(r Model, characterId uint32) error {
		return func(r Model, characterId uint32) error {
			// Emit TRIGGER command to atlas-reactor-actions for script processing
			err := producer.ProviderImpl(l)(ctx)(EnvCommandReactorActionsTopic)(triggerActionsCommandProvider(r, characterId))
			if err != nil {
				l.WithError(err).Warnf("Failed to emit TRIGGER command to reactor-actions for reactor [%d].", r.Id())
				// Don't fail - continue with destruction
			}

			return Destroy(l)(ctx)(r)
		}
	}
}

func containsSkill(skills []uint32, skillId uint32) bool {
	for _, s := range skills {
		if s == skillId {
			return true
		}
	}
	return false
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
