package projection_test

import (
	"atlas-character-factory/configuration/projection"
	"atlas-character-factory/configuration/tenant"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestDecodeTenantEnvelope_ParsesShape(t *testing.T) {
	id := uuid.New()
	bts, err := json.Marshal(map[string]any{
		"schema_version": 1,
		"id":             id.String(),
		"config":         map[string]any{"region": "GMS"},
		"emitted_at":     "2026-06-12T12:00:00Z",
	})
	require.NoError(t, err)
	env, err := projection.DecodeTenantEnvelope(bts)
	require.NoError(t, err)
	require.Equal(t, 1, env.SchemaVersion)
	require.Equal(t, id.String(), env.Id)
	require.NotNil(t, env.Config)
}

func TestDecodeTenantEnvelope_RejectsFutureSchema(t *testing.T) {
	bts, _ := json.Marshal(map[string]any{
		"schema_version": projection.SupportedSchemaVersion + 1,
		"id":             uuid.New().String(),
		"config":         map[string]any{},
	})
	_, err := projection.DecodeTenantEnvelope(bts)
	require.ErrorIs(t, err, projection.ErrUnsupportedSchema)
}

func TestIsTombstone(t *testing.T) {
	require.True(t, projection.IsTombstone(nil))
	require.False(t, projection.IsTombstone([]byte("{}")))
}

func TestState_ApplyAndSnapshot_SetsId(t *testing.T) {
	s := projection.NewState()

	tid := uuid.New()
	trm := tenant.RestModel{Region: "GMS", MajorVersion: 84, MinorVersion: 1}
	trmBts, _ := json.Marshal(trm)
	require.NoError(t, s.ApplyTenant(projection.TenantEnvelope{
		SchemaVersion: 1, Id: tid.String(), Config: trmBts,
	}))

	tenants := s.Snapshot()
	require.Len(t, tenants, 1)
	require.Equal(t, "GMS", tenants[tid].Region)
	// Id is json:"-" in the payload; ApplyTenant must populate it from env.Id.
	require.Equal(t, tid.String(), tenants[tid].Id)

	// Snapshot returns a copy: mutating it does not affect State.
	delete(tenants, tid)
	require.Len(t, s.Snapshot(), 1)

	s.ApplyTenantTombstone(tid)
	require.Empty(t, s.Snapshot())
}

func TestApplyTenant_RejectsBadId(t *testing.T) {
	s := projection.NewState()
	err := s.ApplyTenant(projection.TenantEnvelope{
		SchemaVersion: 1, Id: "not-a-uuid", Config: json.RawMessage(`{"region":"GMS"}`),
	})
	require.Error(t, err)
}

func TestCaughtUp_TransitionsAndUnblocksWaiters(t *testing.T) {
	c := projection.NewCaughtUp()
	require.False(t, c.CaughtUpNow())

	c.SetEndOffsets("T1", map[int]int64{0: 3})
	require.False(t, c.CaughtUpNow())

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	waitDone := make(chan error, 1)
	go func() { waitDone <- c.WaitCaughtUp(ctx) }()

	c.Observe("T1", 0, 1)
	require.False(t, c.CaughtUpNow())
	c.Observe("T1", 0, 2)
	require.True(t, c.CaughtUpNow())

	require.NoError(t, <-waitDone)

	// One-way: a lower observation does not un-flip the gate.
	c.Observe("T1", 0, 0)
	require.True(t, c.CaughtUpNow())
}

func TestCaughtUp_EmptyTopicTriviallyCaughtUp(t *testing.T) {
	c := projection.NewCaughtUp()
	c.SetEndOffsets("T", map[int]int64{})
	require.True(t, c.CaughtUpNow())
}

func TestCaughtUp_EndOffsetOneRequiresObservation(t *testing.T) {
	c := projection.NewCaughtUp()
	c.SetEndOffsets("T", map[int]int64{0: 1})
	require.False(t, c.CaughtUpNow())
	c.Observe("T", 0, 0)
	require.True(t, c.CaughtUpNow())
}

func TestCaughtUp_EmptyPartitionTriviallyCaughtUp(t *testing.T) {
	c := projection.NewCaughtUp()
	c.SetEndOffsets("T", map[int]int64{0: 0})
	require.True(t, c.CaughtUpNow())
}

func TestProjectionDecodesMapleLife(t *testing.T) {
	s := projection.NewState()

	t.Run("block present", func(t *testing.T) {
		tid := uuid.New()
		cfgBts, err := json.Marshal(map[string]any{
			"region": "GMS",
			"mapleLife": map[string]any{
				"looks": []map[string]any{
					{
						"gender":     0,
						"faces":      []uint32{20000},
						"hairs":      []uint32{30000},
						"hairColors": []uint32{0},
						"skinColors": []uint32{0},
					},
				},
				"classes": []map[string]any{
					{
						"ordinal":   0,
						"gender":    0,
						"jobId":     100,
						"level":     10,
						"mapId":     10000,
						"stats":     map[string]any{"str": 4, "dex": 4, "int": 4, "luk": 4, "hp": 50, "mp": 5},
						"ap":        0,
						"sp":        "0,0,0,0,0,0,0,0,0,0",
						"spSkillId": 1000001,
						"meso":      0,
						"equipment": []map[string]any{},
						"inventory": []map[string]any{},
					},
				},
			},
		})
		require.NoError(t, err)

		require.NoError(t, s.ApplyTenant(projection.TenantEnvelope{
			SchemaVersion: 1, Id: tid.String(), Config: cfgBts,
		}))

		snap := s.Snapshot()
		require.Len(t, snap[tid].MapleLife.Classes, 1)
		require.Equal(t, uint32(100), snap[tid].MapleLife.Classes[0].JobId)
		require.Equal(t, uint32(1000001), snap[tid].MapleLife.Classes[0].SpSkillId)
		require.Equal(t, []uint32{20000}, snap[tid].MapleLife.Looks[0].Faces)
	})

	t.Run("block absent", func(t *testing.T) {
		tid := uuid.New()
		cfgBts, err := json.Marshal(map[string]any{"region": "GMS"})
		require.NoError(t, err)

		require.NoError(t, s.ApplyTenant(projection.TenantEnvelope{
			SchemaVersion: 1, Id: tid.String(), Config: cfgBts,
		}))

		snap := s.Snapshot()
		require.Empty(t, snap[tid].MapleLife.Classes)
	})
}
