// Package occurrence is the generic persistence for a live instance of an
// event: one row per occurrence, its map scope and its monster set, paired
// with the transition package's history (Task 14). The two structural rules
// enforced here are what make FR-O6/FR-T2 and FR-B20 hold as database
// predicates rather than by caller discipline:
//
//  1. Every state or stage change writes the occurrence row and a transition
//     row in ONE transaction. Processor exposes only the paired operation.
//  2. Completion is a guarded UPDATE ("WHERE id = ? AND state = 'ACTIVE'"),
//     not a lock: RowsAffected == 0 means another path already completed it.
package occurrence

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// Occurrence states.
const (
	StateActive    = "ACTIVE"
	StateCompleted = "COMPLETED"
	StateCancelled = "CANCELLED"
	StateFailed    = "FAILED"
)

// ReasonMonstersEliminated is the CompletionReason for an occurrence whose
// monster SET (MonsterTally) reached "every spawned monster accounted for
// and none alive" (FR-B18). Declared here, not in a specific event type's
// package, because the monster-tracking primitives it completes
// (ObserveMonsterSpawned/ObserveMonsterGone/MonsterTally) are themselves
// generic to this package — any event type that spawns monsters shares this
// completion reason rather than inventing its own string.
const ReasonMonstersEliminated = "MONSTERS_ELIMINATED"

// ReasonVesselArrived is the CompletionReason for an occurrence completed
// because the voyage it was scoped to reached its destination
// (transport.EventStatusVoyageArrived) before every spawned monster was
// eliminated. Declared here, beside ReasonMonstersEliminated, for the same
// reason: it names one of the (possibly several) generic ways any
// voyage-scoped occurrence can complete, not something specific to one
// event type's package.
const ReasonVesselArrived = "VESSEL_ARRIVED"

// Model is an immutable representation of a single live event occurrence.
type Model struct {
	id               uuid.UUID
	definitionId     uuid.UUID
	theType          string
	state            string
	stage            string
	context          json.RawMessage
	worldId          world.Id
	channelId        channel.Id
	voyageId         uuid.UUID
	concurrencyKey   string
	startedAt        time.Time
	nextTransitionAt *time.Time
	completedAt      *time.Time
	completionReason string
}

func (m Model) Id() uuid.UUID                { return m.id }
func (m Model) DefinitionId() uuid.UUID      { return m.definitionId }
func (m Model) Type() string                 { return m.theType }
func (m Model) State() string                { return m.state }
func (m Model) Stage() string                { return m.stage }
func (m Model) Context() json.RawMessage     { return m.context }
func (m Model) WorldId() world.Id            { return m.worldId }
func (m Model) ChannelId() channel.Id        { return m.channelId }
func (m Model) VoyageId() uuid.UUID          { return m.voyageId }
func (m Model) ConcurrencyKey() string       { return m.concurrencyKey }
func (m Model) StartedAt() time.Time         { return m.startedAt }
func (m Model) NextTransitionAt() *time.Time { return m.nextTransitionAt }
func (m Model) CompletedAt() *time.Time      { return m.completedAt }
func (m Model) CompletionReason() string     { return m.completionReason }
