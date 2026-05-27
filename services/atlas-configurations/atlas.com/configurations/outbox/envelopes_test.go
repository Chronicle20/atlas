package outbox_test

import (
	"encoding/json"
	"testing"
	"time"

	"atlas-configurations/outbox"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestNewServiceEnvelope_Shape(t *testing.T) {
	id := uuid.New()
	cfg := map[string]any{"type": "channel-service", "name": "ch-1"}
	emittedAt := time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)

	b, err := outbox.NewServiceEnvelope(id, cfg, emittedAt)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(b, &got))
	require.Equal(t, float64(outbox.CurrentSchemaVersion), got["schema_version"])
	require.Equal(t, id.String(), got["id"])
	require.NotNil(t, got["config"])
	require.Equal(t, "2026-05-17T12:00:00Z", got["emitted_at"])
}

func TestNewTenantEnvelope_MatchesServiceShape(t *testing.T) {
	id := uuid.New()
	cfg := map[string]any{"region": "GMS"}
	emittedAt := time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)

	s, err := outbox.NewServiceEnvelope(id, cfg, emittedAt)
	require.NoError(t, err)
	t1, err := outbox.NewTenantEnvelope(id, cfg, emittedAt)
	require.NoError(t, err)

	require.JSONEq(t, string(s), string(t1))
}
