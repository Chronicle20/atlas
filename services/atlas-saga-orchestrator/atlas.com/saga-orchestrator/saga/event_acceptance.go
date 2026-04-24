package saga

import (
	"github.com/sirupsen/logrus"

	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

// EventKind is a compact tag for the semantic class of an event received on a
// status topic. Each consumer handler hardcodes one EventKind constant per
// handler function; the handler's identity is the classification.
type EventKind string

const (
	// Character subsystem.
	EventKindCharacterMapChanged        EventKind = "character.map_changed"
	EventKindCharacterExperienceChanged EventKind = "character.experience_changed"
	EventKindCharacterLevelChanged      EventKind = "character.level_changed"
	EventKindCharacterMesoChanged       EventKind = "character.meso_changed"
	EventKindCharacterJobChanged        EventKind = "character.job_changed"
	EventKindCharacterCreated           EventKind = "character.created"
	EventKindCharacterCreationFailed    EventKind = "character.creation_failed"
	EventKindCharacterStatChanged       EventKind = "character.stat_changed"
	EventKindCharacterMesoError         EventKind = "character.meso_error"
	EventKindCharacterDeleted           EventKind = "character.deleted"

	// Asset subsystem.
	EventKindAssetCreated         EventKind = "asset.created"
	EventKindAssetDeleted         EventKind = "asset.deleted"
	EventKindAssetQuantityChanged EventKind = "asset.quantity_changed"
	EventKindAssetMoved           EventKind = "asset.moved"
)

// acceptanceTable maps each saga.Action to the set of EventKinds that can
// complete (or fail) a step of that action. Empty slice means self-completing
// — no Kafka event advances the step. A missing entry is a bug: unknown
// actions default-deny in StepAcceptsEvent, but the coverage test
// (event_acceptance_test.go) catches missing entries before runtime.
//
// Task 1 only seeds the three actions from the §9.1 Thief scenario.
// Task 2 fills the remaining entries.
var acceptanceTable = map[sharedsaga.Action][]EventKind{
	sharedsaga.RebalanceAP: {EventKindCharacterStatChanged},
	sharedsaga.ChangeJob:   {EventKindCharacterJobChanged},
	sharedsaga.AwardAsset:  {EventKindAssetCreated, EventKindAssetQuantityChanged},
}

// StepAcceptsEvent reports whether a saga step's Action can be legitimately
// completed by an event of the given EventKind. Unknown actions default-deny.
func StepAcceptsEvent(action sharedsaga.Action, kind EventKind) bool {
	kinds, ok := acceptanceTable[action]
	if !ok {
		return false
	}
	for _, k := range kinds {
		if k == kind {
			return true
		}
	}
	return false
}

// skipReason* constants are the `reason` field values on structured debug
// logs emitted when AcceptEvent (or a handler-level guard) refuses to
// complete a step. Centralised so per-consumer drift is impossible.
const (
	skipReasonSagaNotFound       = "saga_not_found"
	skipReasonNoPendingStep      = "no_pending_step"
	skipReasonActionMismatch     = "action_mismatch"
	skipReasonTemplateIdMismatch = "template_id_mismatch"
	skipReasonUnmatchedEvent     = "unmatched_event"
)

// logSkip emits a debug-level structured log with a `reason` field.
// Consumer packages may call this directly for handler-local skips
// (e.g., template-id mismatch in the asset handler).
func logSkip(l logrus.FieldLogger, fields logrus.Fields, reason string) {
	if fields == nil {
		fields = logrus.Fields{}
	}
	fields["reason"] = reason
	l.WithFields(fields).Debug("Saga event skipped.")
}
