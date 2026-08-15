// Package orchestration is a thin composition point between event/definition
// and event/scheduling (task-231 R33-3, pre-flight ruling F1).
//
// event/definition must not import event/scheduling: event/scheduling already
// imports event/definition (its dispatch() resolves a definition per claimed
// row), so the reverse import would cycle. This package imports both and owns
// the false->true "enabling schedules work" behavior (FR-A2) so that
// event/definition.Processor.SetEnabled stays toggle-only, and so the
// business logic does not live in the REST PATCH handler (FR-N18).
//
// The scheduled row itself is entirely generic — one TRIGGER_EVALUATION at
// time.Now(), empty work context, dedupe key enable:<definitionId> — so this
// package needs no knowledge of which event type was enabled (FR-X3). What
// an empty-context evaluation MEANS is decided entirely by the handler's own
// Evaluate (registry.Get(d.Type()).Evaluate), resolved through the registry
// exactly like every other TRIGGER_EVALUATION row.
package orchestration

import (
	"atlas-events/event/definition"
	"atlas-events/event/scheduling"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// SetEnabled toggles definition id's enabled flag via definition.Processor
// and, on a false->true transition, schedules the one generic
// TRIGGER_EVALUATION row FR-A2 requires. A false->false, true->true or
// true->false transition schedules nothing — disabling a definition must
// never touch its work (definition.Processor.SetEnabled's own FR-D5
// contract), and re-enabling an already-enabled definition is a no-op PATCH,
// not a re-trigger.
func SetEnabled(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) func(id uuid.UUID, enabled bool) (definition.Model, error) {
	return func(id uuid.UUID, enabled bool) (definition.Model, error) {
		dp := definition.NewProcessor(l, ctx, db)

		before, err := dp.GetById(id)
		if err != nil {
			return definition.Model{}, err
		}

		updated, err := dp.SetEnabled(id, enabled)
		if err != nil {
			return definition.Model{}, err
		}

		if !before.Enabled() && enabled {
			m, err := scheduling.NewBuilder(updated.Id(), scheduling.WorkTypeTriggerEvaluation).
				SetContext(json.RawMessage(`{}`)).
				SetExecuteAt(time.Now()).
				SetDedupeKey(fmt.Sprintf("enable:%s", updated.Id())).
				Build()
			if err != nil {
				return definition.Model{}, err
			}
			if _, _, err := scheduling.NewAdministrator(l, ctx, db).Schedule(m); err != nil {
				return definition.Model{}, err
			}
		}

		return updated, nil
	}
}
