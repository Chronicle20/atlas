// Package env carries the execution environment of an operation on
// context.Context, exactly as libs/atlas-tenant carries the tenant.
//
// Environment is a property of the OPERATION, not of the deployment
// processing it. A baseline pod may process an operation belonging to an
// ephemeral environment; it reads the environment off the operation and
// hands it back out unchanged.
//
// This package is deliberately a leaf: it depends only on atlas-model, so
// atlas-kafka and atlas-rest can import it without a module cycle. The
// registry implementation that this package's Registry interface describes
// is populated in libs/atlas-service, which owns the Kafka projection.
package env

import (
	"context"
	"os"
	"regexp"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// Key is the REST/Kafka header name for the environment id, matching
// libs/atlas-tenant's ID = "TENANT_ID" convention. Its string value is part
// of the wire contract; it must not change.
const Key = "ENVIRONMENT"

// SelfVar is the environment variable a pod reads its own environment from.
const SelfVar = "ATLAS_ENVIRONMENT"

// ctxKey is a private context-key type so this package's context value can
// never collide with another package's, including one that also happens to
// store a "ENVIRONMENT" string key (staticcheck SA1029). Key's string value
// remains the wire header name; ctxKeyEnv is only ever used as the
// context.Context key, never serialized.
type ctxKey string

const ctxKeyEnv ctxKey = Key

// Id identifies one execution environment ("main", "pr-123"). The empty Id
// is the legacy value: it means "not environment-aware" and every registry
// query answers it with the local deployment (FR-1.8).
type Id string

// idPattern constrains ids at INGEST only (task-232 P2). Operations are
// never revalidated: a record that entered the registry is trusted, and a
// per-operation regex would be I/O-free but pointless work on the hot path.
var idPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,30}[a-z0-9]$`)

// Valid reports whether id is a well-formed environment id. The empty id is
// valid — it is the legacy value.
func Valid(id Id) bool {
	if id == "" {
		return true
	}
	return idPattern.MatchString(string(id))
}

func WithContext(ctx context.Context, id Id) context.Context {
	return context.WithValue(ctx, ctxKeyEnv, id)
}

// FromContext never errors: a context with no environment yields the empty
// id, which is the legacy value (FR-1.8). It returns a Provider anyway so
// callers compose with the rest of the atlas-model pipeline.
func FromContext(ctx context.Context) model.Provider[Id] {
	id, _ := ctx.Value(ctxKeyEnv).(Id)
	return model.FixedProvider(id)
}

func MustFromContext(ctx context.Context) Id {
	id, _ := FromContext(ctx)()
	return id
}

// Self is this process's own environment, read from ATLAS_ENVIRONMENT. It
// never fails and never consults the registry: a pod knows its own
// environment even during a registry outage, which is what keeps main fully
// functional when the projection is unavailable (design §4.3).
func Self() Id {
	return Id(os.Getenv(SelfVar))
}
