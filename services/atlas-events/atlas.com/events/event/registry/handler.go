// Package registry is the single seam between the generic event infrastructure
// and event-specific behavior (FR-X3). The generic layer reaches event behavior
// ONLY through Handler, resolved by type string. It must never switch on a type
// constant — Task 39's AST test enforces that.
//
// Domain reactions (a monster died, a voyage arrived, a character logged in) do
// NOT go through this interface. They are ordinary Kafka consumers owned by the
// event's own package and registered from that package's InitConsumers. This is
// the difference between "a registry mapping type to a handler" (allowed) and
// "a central dispatch table containing event logic" (forbidden).
package registry

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// Seed is a handler's request to create an occurrence. The generic layer owns
// how it becomes rows.
type Seed struct {
	Stage            string
	Context          json.RawMessage
	WorldId          world.Id
	ChannelId        channel.Id
	VoyageId         uuid.UUID // uuid.Nil when the event has no voyage scope
	ConcurrencyKey   string
	Maps             []MapScope
	NextTransitionAt *time.Time
}

type MapScope struct {
	MapId  _map.Id
	Visual bool
}

// Progress is what a handler settles an occurrence into after Start/Advance.
// Terminal == true completes the occurrence with CompletionReason.
type Progress struct {
	Stage            string
	NextTransitionAt *time.Time
	Terminal         bool
	CompletionReason string
}

// Definition and Occurrence are the read-only views a handler receives.
type Definition struct {
	Id            uuid.UUID
	Type          string
	Name          string
	Enabled       bool
	Configuration json.RawMessage
}

type Occurrence struct {
	Id           uuid.UUID
	DefinitionId uuid.UUID
	Type         string
	Stage        string
	Context      json.RawMessage
	WorldId      world.Id
	ChannelId    channel.Id
	VoyageId     uuid.UUID
	StartedAt    time.Time
}

// Work is the due scheduled row that drove the call.
type Work struct {
	Id      uuid.UUID
	Type    string
	Context json.RawMessage
}

type Handler interface {
	// Type is the definition type this handler serves. Used as the registry key.
	Type() string

	// ValidateConfiguration rejects a definition whose configuration this handler
	// cannot interpret (FR-D6); returns a field-scoped error.
	ValidateConfiguration(raw json.RawMessage) error

	// ConcurrencyKey names the gameplay slot an occurrence would occupy. The
	// generic layer enforces at most one non-terminal occurrence per
	// (tenant, definition, key). Empty string means unlimited.
	ConcurrencyKey(ctx context.Context, workContext json.RawMessage) (string, error)

	// ConcurrencyKeyIsConstant reports whether ConcurrencyKey's result never
	// varies with workContext — i.e. this handler's occurrences are scoped to
	// a single, per-type gameplay slot rather than one per voyage/world/
	// channel/etc. This cannot be discovered by probing ConcurrencyKey with
	// distinct payloads and comparing (FR-UI4's original approach,
	// event/definition/processor.go's singleOccurrence): json.Unmarshal
	// silently ignores unknown fields, so a handler that decodes the probe
	// into a typed struct (e.g. CRIMSON_BALROG's WorkContext) gets the SAME
	// zero value back for every unrecognized probe payload — collapsing two
	// distinct probes to equal keys and misreporting a varying handler as
	// constant (task-231 R33-4). The handler must simply declare the answer.
	ConcurrencyKeyIsConstant() bool

	// Evaluate decides whether a TRIGGER_EVALUATION should produce an
	// occurrence. Returning (nil, nil) is the ordinary "no occurrence" outcome
	// (FR-B7, FR-B8), not an error.
	Evaluate(ctx context.Context, d Definition, w Work) (*Seed, error)

	// Start orchestrates the side effects of a newly created occurrence (FR-B11).
	Start(ctx context.Context, o Occurrence) (Progress, error)

	// Advance handles a due OCCURRENCE_TRANSITION row (FR-A14). Completion side
	// effects (FR-B18, FR-B19, FR-A15) belong here too, on the terminal branch —
	// there is no separate Complete method: both real implementations converge
	// on occurrence.Processor.Complete directly, and a Handler.Complete would
	// have zero production callers.
	Advance(ctx context.Context, o Occurrence, w Work) (Progress, error)
}
