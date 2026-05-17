package projection_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"atlas-channel/configuration"
	"atlas-channel/configuration/projection"
	"atlas-channel/configuration/tenant"
	"atlas-channel/server"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestDecodeServiceEnvelope_ParsesShape(t *testing.T) {
	id := uuid.New()
	bts, err := json.Marshal(map[string]any{
		"schema_version": 1,
		"id":             id.String(),
		"config":         map[string]any{"tenants": []any{}},
		"emitted_at":     "2026-05-17T12:00:00Z",
	})
	require.NoError(t, err)
	env, err := projection.DecodeServiceEnvelope(bts)
	require.NoError(t, err)
	require.Equal(t, 1, env.SchemaVersion)
	require.Equal(t, id.String(), env.Id)
	require.NotNil(t, env.Config)
}

func TestDecodeServiceEnvelope_RejectsFutureSchema(t *testing.T) {
	bts, _ := json.Marshal(map[string]any{
		"schema_version": projection.SupportedSchemaVersion + 1,
		"id":             uuid.New().String(),
		"config":         map[string]any{},
	})
	_, err := projection.DecodeServiceEnvelope(bts)
	require.ErrorIs(t, err, projection.ErrUnsupportedSchema)
}

func TestIsTombstone(t *testing.T) {
	require.True(t, projection.IsTombstone(nil))
	require.False(t, projection.IsTombstone([]byte("{}")))
}

func TestState_ApplyAndSnapshot(t *testing.T) {
	s := projection.NewState()

	svcId := uuid.New()
	cfg := configuration.RestModel{
		Tenants: []configuration.ChannelTenantRestModel{
			{Id: uuid.New().String(), IPAddress: "10.0.0.1"},
		},
	}
	cfgBts, _ := json.Marshal(cfg)
	require.NoError(t, s.ApplyService(projection.ServiceEnvelope{
		SchemaVersion: 1, Id: svcId.String(), Config: cfgBts,
	}))

	tid := uuid.New()
	trm := tenant.RestModel{Region: "GMS", MajorVersion: 83, MinorVersion: 1}
	trmBts, _ := json.Marshal(trm)
	require.NoError(t, s.ApplyTenant(projection.TenantEnvelope{
		SchemaVersion: 1, Id: tid.String(), Config: trmBts,
	}))

	svc, tenants := s.Snapshot()
	require.NotNil(t, svc)
	require.Equal(t, svcId, svc.Id)
	require.Len(t, svc.Tenants, 1)
	require.Equal(t, "GMS", tenants[tid].Region)

	s.ApplyTenantTombstone(tid)
	_, tenants = s.Snapshot()
	require.Empty(t, tenants)

	s.ApplyServiceTombstone()
	svc, _ = s.Snapshot()
	require.Nil(t, svc)
}

func TestComputeOps_AddRemovePortChangeUnchanged(t *testing.T) {
	tid := uuid.New()
	tcfg := map[uuid.UUID]tenant.RestModel{
		tid: {Region: "GMS", MajorVersion: 83, MinorVersion: 1},
	}

	mk := func(port int) *configuration.RestModel {
		return &configuration.RestModel{
			Tenants: []configuration.ChannelTenantRestModel{{
				Id:        tid.String(),
				IPAddress: "10.0.0.1",
				Worlds: []configuration.ChannelWorldRestModel{{
					Id: 1,
					Channels: []configuration.ChannelChannelRestModel{
						{Id: 0, Port: port},
					},
				}},
			}},
		}
	}
	key := server.Key{TenantId: tid, WorldId: world.Id(1), ChannelId: channel.Id(0)}

	// ADD: empty → one channel
	ops := projection.ComputeOps(nil, nil, mk(8585), tcfg)
	require.Len(t, ops, 1)
	require.Equal(t, projection.OpAdd, ops[0].Kind)
	require.Equal(t, key, ops[0].Key)
	require.Equal(t, 8585, ops[0].Cfg.Port)

	// UNCHANGED: same → no ops
	ops = projection.ComputeOps(mk(8585), tcfg, mk(8585), tcfg)
	require.Empty(t, ops)

	// PORT CHANGE: drain then add
	ops = projection.ComputeOps(mk(8585), tcfg, mk(9090), tcfg)
	require.Len(t, ops, 2)
	var sawDrain, sawAdd bool
	for _, op := range ops {
		switch op.Kind {
		case projection.OpDrain:
			sawDrain = true
			require.Equal(t, key, op.Key)
		case projection.OpAdd:
			sawAdd = true
			require.Equal(t, 9090, op.Cfg.Port)
		}
	}
	require.True(t, sawDrain && sawAdd)

	// REMOVE: present → absent → drain
	ops = projection.ComputeOps(mk(8585), tcfg, nil, nil)
	require.Len(t, ops, 1)
	require.Equal(t, projection.OpDrain, ops[0].Kind)

	// TENANT MISSING: service references tenant not in tenantConfigs → skipped
	ops = projection.ComputeOps(nil, nil, mk(8585), nil)
	require.Empty(t, ops, "tenant config missing → no Add op")
}

func TestCaughtUp_TransitionsAndUnblocksWaiters(t *testing.T) {
	c := projection.NewCaughtUp()
	require.False(t, c.CaughtUpNow())

	// One topic, one partition with end offset 3 → caught-up after observing offset 2.
	c.SetEndOffsets("T1", map[int]int64{0: 3})
	require.False(t, c.CaughtUpNow())

	// Empty topic counts as already caught-up.
	c.SetEndOffsets("T2", map[int]int64{})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	waitDone := make(chan error, 1)
	go func() { waitDone <- c.WaitCaughtUp(ctx) }()

	c.Observe("T1", 0, 1)
	require.False(t, c.CaughtUpNow())
	c.Observe("T1", 0, 2)
	require.True(t, c.CaughtUpNow())

	require.NoError(t, <-waitDone)

	// One-way: feeding a lower observation doesn't un-flip the gate.
	c.Observe("T1", 0, 0)
	require.True(t, c.CaughtUpNow())
}

func TestCaughtUp_ReadyChecker(t *testing.T) {
	c := projection.NewCaughtUp()
	fn := c.ReadyChecker()
	require.False(t, fn())
	c.SetEndOffsets("T", map[int]int64{})
	require.True(t, fn())
}
