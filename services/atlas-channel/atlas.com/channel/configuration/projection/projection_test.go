package projection_test

import (
	"atlas-channel/configuration"
	"atlas-channel/configuration/projection"
	"atlas-channel/configuration/tenant"
	"atlas-channel/listener"
	"atlas-channel/server"
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenantpkg "github.com/Chronicle20/atlas/libs/atlas-tenant"
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

// Regression: a partition with end-offset 1 (one pending record) must NOT
// be considered caught up until offset 0 has been observed. Previously
// the default int64 zero returned by `got[p]` for a never-observed
// partition satisfied the `>= end-1` check, marking the gate caught up
// before the record was consumed.
func TestCaughtUp_EndOffsetOneRequiresObservation(t *testing.T) {
	c := projection.NewCaughtUp()
	c.SetEndOffsets("T", map[int]int64{0: 1})
	require.False(t, c.CaughtUpNow(), "end=1 with no observation must not count as caught up")

	c.Observe("T", 0, 0)
	require.True(t, c.CaughtUpNow(), "end=1 with offset 0 observed must count as caught up")
}

// Regression: a partition with end-offset 0 (empty partition) is
// trivially caught up — the empty-topic semantics must extend to
// single-partition empties returned by Kafka.
func TestCaughtUp_EmptyPartitionTriviallyCaughtUp(t *testing.T) {
	c := projection.NewCaughtUp()
	c.SetEndOffsets("T", map[int]int64{0: 0})
	require.True(t, c.CaughtUpNow(), "end=0 must count as already caught up")
}

// FR-1.5/D6: HasService is the readiness signal — a pod whose own
// service-config row never arrives must never report Ready. The
// "different service's row" case is enforced one layer up: subscriber.go
// returns early on env.Id != s.ServiceId.String() (subscriber.go:112),
// so a foreign row never reaches State and is not re-tested here.

func TestHasServiceIsFalseBeforeAnyServiceIsApplied(t *testing.T) {
	s := projection.NewState()
	require.False(t, s.HasService())
}

func TestHasServiceIsTrueAfterTheMatchingServiceIsApplied(t *testing.T) {
	s := projection.NewState()
	cfg := configuration.RestModel{}
	cfgBts, _ := json.Marshal(cfg)
	require.NoError(t, s.ApplyService(projection.ServiceEnvelope{
		SchemaVersion: 1,
		Id:            "5a86d8e6-3167-5e74-9fc5-021d94001da2",
		Config:        cfgBts,
	}))
	require.True(t, s.HasService())
}

func TestHasServiceIsFalseAgainAfterATombstone(t *testing.T) {
	s := projection.NewState()
	cfg := configuration.RestModel{}
	cfgBts, _ := json.Marshal(cfg)
	require.NoError(t, s.ApplyService(projection.ServiceEnvelope{
		SchemaVersion: 1,
		Id:            "5a86d8e6-3167-5e74-9fc5-021d94001da2",
		Config:        cfgBts,
	}))
	require.True(t, s.HasService())

	s.ApplyServiceTombstone()
	require.False(t, s.HasService())
}

func TestHasServiceIsFalseAfterOnlyATenantIsApplied(t *testing.T) {
	s := projection.NewState()
	trm := tenant.RestModel{Region: "GMS", MajorVersion: 83, MinorVersion: 1}
	trmBts, _ := json.Marshal(trm)
	require.NoError(t, s.ApplyTenant(projection.TenantEnvelope{
		SchemaVersion: 1,
		Id:            uuid.New().String(),
		Config:        trmBts,
	}))
	require.False(t, s.HasService())
}

// applyOneChannel puts exactly one (tid, w, c) channel into s, so the next
// ComputeOps diff against an empty prev emits a single OpAdd for key.
func applyOneChannel(t *testing.T, s *projection.State, tid uuid.UUID, w world.Id, c channel.Id, port int) {
	cfg := configuration.RestModel{
		Tenants: []configuration.ChannelTenantRestModel{{
			Id:        tid.String(),
			IPAddress: "127.0.0.1",
			Worlds: []configuration.ChannelWorldRestModel{{
				Id: byte(w),
				Channels: []configuration.ChannelChannelRestModel{
					{Id: byte(c), Port: port},
				},
			}},
		}},
	}
	cfgBts, err := json.Marshal(cfg)
	require.NoError(t, err)
	require.NoError(t, s.ApplyService(projection.ServiceEnvelope{
		SchemaVersion: 1, Id: uuid.New().String(), Config: cfgBts,
	}))

	trm := tenant.RestModel{Region: "GMS", MajorVersion: 83, MinorVersion: 1}
	trmBts, err := json.Marshal(trm)
	require.NoError(t, err)
	require.NoError(t, s.ApplyTenant(projection.TenantEnvelope{
		SchemaVersion: 1, Id: tid.String(), Config: trmBts,
	}))
}

// stubServerModel builds the ServerModelFn the apply loop needs to call
// listener.Registry.Add. It mirrors listener/registry_test.go's
// makeServerModel, deriving the tenant.Model and channel.Model from the Op
// fields rather than a fixed value, since ComputeOps produces the key.
func stubServerModel(key server.Key, cfg projection.ListenerConfig) server.Model {
	tm, _ := tenantpkg.Create(key.TenantId, cfg.Region, cfg.MajorVersion, cfg.MinorVersion)
	return server.NewProcessor(logrus.New(), context.Background()).
		Register(tm, channel.NewModel(key.WorldId, key.ChannelId), cfg.IPAddress, cfg.Port)
}

func nullLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)
	return l
}

func TestApplyLoop_AddBodyReceivesAContextCanceledByDrain(t *testing.T) {
	tid := uuid.New()
	key := server.Key{TenantId: tid, WorldId: world.Id(1), ChannelId: channel.Id(0)}
	defer server.GetRegistry().Deregister(key)

	got := make(chan context.Context, 1)

	s := projection.NewState()
	applyOneChannel(t, s, tid, key.WorldId, key.ChannelId, 8585)

	c := projection.NewCaughtUp()
	c.SetEndOffsets("T", map[int]int64{})

	registry := listener.NewRegistry(nullLogger(), listener.Dependencies{
		UnregisterChannel: func(channel.Model) error { return nil },
		RemoveHandler:     func(string, string) error { return nil },
	}, listener.Config{})

	loop := &projection.ApplyLoop{
		State:    s,
		CaughtUp: c,
		Registry: registry,
		AddBody: func(parent context.Context, key server.Key, cfg projection.ListenerConfig, h *listener.Handle) ([]listener.HandlerHandle, error) {
			select {
			case got <- parent:
			default:
			}
			return nil, nil
		},
		ServerModel: stubServerModel,
		Interval:    10 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go loop.Run(ctx, nullLogger())

	var parent context.Context
	select {
	case parent = <-got:
	case <-time.After(2 * time.Second):
		t.Fatal("AddBody was never called")
	}
	require.NoError(t, parent.Err())

	require.NoError(t, registry.Drain(key))

	// This is the assertion that pins defect 1: with loop.go still
	// passing the apply loop's own ctx into AddBody, the captured ctx is
	// still live (derived from ctx, canceled only at ctx.Done()), and
	// this fails.
	require.Equal(t, context.Canceled, parent.Err())
}

func TestApplyLoop_RetriesAFailedAddOnTheNextTick(t *testing.T) {
	tid := uuid.New()
	key := server.Key{TenantId: tid, WorldId: world.Id(1), ChannelId: channel.Id(0)}
	defer server.GetRegistry().Deregister(key)

	s := projection.NewState()
	applyOneChannel(t, s, tid, key.WorldId, key.ChannelId, 8586)

	c := projection.NewCaughtUp()
	c.SetEndOffsets("T", map[int]int64{})

	registry := listener.NewRegistry(nullLogger(), listener.Dependencies{
		UnregisterChannel: func(channel.Model) error { return nil },
		RemoveHandler:     func(string, string) error { return nil },
	}, listener.Config{})

	var calls atomic.Int32
	loop := &projection.ApplyLoop{
		State:    s,
		CaughtUp: c,
		Registry: registry,
		AddBody: func(parent context.Context, key server.Key, cfg projection.ListenerConfig, h *listener.Handle) ([]listener.HandlerHandle, error) {
			if calls.Add(1) == 1 {
				return nil, errors.New("bind failed")
			}
			return nil, nil
		},
		ServerModel: stubServerModel,
		Interval:    10 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go loop.Run(ctx, nullLogger())

	deadline := time.Now().Add(2 * time.Second)
	for calls.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	require.GreaterOrEqual(t, calls.Load(), int32(2), "the failed add must be retried on a later tick")

	_, ok := registry.Get(key)
	require.True(t, ok, "the retried add must have succeeded and registered the handle")
}

func TestApplyLoop_DropsAPendingAddWhenTheKeyLeavesConfig(t *testing.T) {
	tid := uuid.New()
	key := server.Key{TenantId: tid, WorldId: world.Id(1), ChannelId: channel.Id(0)}
	defer server.GetRegistry().Deregister(key)

	s := projection.NewState()
	applyOneChannel(t, s, tid, key.WorldId, key.ChannelId, 8587)

	c := projection.NewCaughtUp()
	c.SetEndOffsets("T", map[int]int64{})

	registry := listener.NewRegistry(nullLogger(), listener.Dependencies{
		UnregisterChannel: func(channel.Model) error { return nil },
		RemoveHandler:     func(string, string) error { return nil },
	}, listener.Config{})

	var calls atomic.Int32
	loop := &projection.ApplyLoop{
		State:    s,
		CaughtUp: c,
		Registry: registry,
		AddBody: func(parent context.Context, key server.Key, cfg projection.ListenerConfig, h *listener.Handle) ([]listener.HandlerHandle, error) {
			calls.Add(1)
			return nil, errors.New("bind failed")
		},
		ServerModel: stubServerModel,
		Interval:    10 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go loop.Run(ctx, nullLogger())

	deadline := time.Now().Add(2 * time.Second)
	for calls.Load() < 1 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	require.GreaterOrEqual(t, calls.Load(), int32(1), "the first tick must have attempted the add")

	s.ApplyServiceTombstone()

	countAtTombstone := calls.Load()
	time.Sleep(100 * time.Millisecond) // >= 5 ticks at 10ms
	require.Equal(t, countAtTombstone, calls.Load(), "a pending add for a key no longer in config must not be retried")
}
