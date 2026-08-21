package monster

import (
	"atlas-monsters/monster/information"
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// timerMobMonsterId and timerMobMaxHp are the task-253 timer-fixture: a mob
// with selfDestruction {action: 4, removeAfter: 0, hp: -1} -- no HP
// predicate, so it only ever detonates via the timer.
const (
	timerMobMonsterId = uint32(9300166)
	timerMobMaxHp     = uint32(1000)
)

func timerSelfDestruction() information.SelfDestruction {
	return information.NewSelfDestruction(true, 4, 0, -1)
}

// entriesForTenant filters GetAll's cross-tenant result to the entries
// belonging to ten, so a table-driven test asserting an exact count is not
// coupled to what other tests in this package have left registered.
func entriesForTenant(ctx context.Context, ten tenant.Model) map[MonsterKey]SelfDestructTimerEntry {
	out := make(map[MonsterKey]SelfDestructTimerEntry)
	for mk, e := range GetSelfDestructTimerRegistry().GetAll(ctx) {
		if mk.Tenant.Id() == ten.Id() {
			out[mk] = e
		}
	}
	return out
}

func TestCreateArmsTimerForTimerMob(t *testing.T) {
	GetMonsterRegistry().Clear(context.Background())
	ten := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), ten)

	prevHook := testInformationLookup
	testInformationLookup = func(_ uint32) (information.Model, error) {
		return information.NewModelBuilder().SetSelfDestruction(timerSelfDestruction()).Build(), nil
	}
	t.Cleanup(func() { testInformationLookup = prevHook })

	p := NewProcessor(logrus.New(), ctx)
	m, err := p.Create(testField(), RestModel{MonsterId: timerMobMonsterId})
	require.NoError(t, err)
	t.Cleanup(func() { GetSelfDestructTimerRegistry().Unregister(ctx, ten, m.UniqueId()) })

	entries := entriesForTenant(ctx, ten)
	require.Len(t, entries, 1)

	entry, ok := entries[MonsterKey{Tenant: ten, MonsterId: m.UniqueId()}]
	require.True(t, ok, "expected an entry keyed by the new uniqueId")
	require.Equal(t, byte(4), entry.Action())
	require.Equal(t, timerMobMonsterId, entry.MonsterId())
	require.False(t, entry.FireAt().After(time.Now()), "removeAfter=0 must fire immediately")
}

func TestCreateDoesNotArmTimerForOtherMobs(t *testing.T) {
	tests := []struct {
		name        string
		sd          information.SelfDestruction
		wantEntries int
		wantDelay   time.Duration
	}{
		{
			name:        "no block",
			sd:          information.NewSelfDestruction(false, 0, -1, -1),
			wantEntries: 0,
		},
		{
			name:        "HP threshold only (Boomer)",
			sd:          information.NewSelfDestruction(true, 1, -1, 1800),
			wantEntries: 0,
		},
		{
			name:        "HP threshold and removeAfter",
			sd:          information.NewSelfDestruction(true, 3, 5, 1),
			wantEntries: 0,
		},
		{
			name:        "timer only, removeAfter 30",
			sd:          information.NewSelfDestruction(true, 4, 30, -1),
			wantEntries: 1,
			wantDelay:   30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			GetMonsterRegistry().Clear(context.Background())
			ten := newTestTenant(t)
			ctx := tenant.WithContext(context.Background(), ten)

			prevHook := testInformationLookup
			testInformationLookup = func(_ uint32) (information.Model, error) {
				return information.NewModelBuilder().SetSelfDestruction(tt.sd).Build(), nil
			}
			t.Cleanup(func() { testInformationLookup = prevHook })

			p := NewProcessor(logrus.New(), ctx)
			m, err := p.Create(testField(), RestModel{MonsterId: 9300167})
			require.NoError(t, err)
			t.Cleanup(func() { GetSelfDestructTimerRegistry().Unregister(ctx, ten, m.UniqueId()) })

			entries := entriesForTenant(ctx, ten)
			require.Len(t, entries, tt.wantEntries)

			if tt.wantEntries == 1 {
				entry, ok := entries[MonsterKey{Tenant: ten, MonsterId: m.UniqueId()}]
				require.True(t, ok)
				wantFireAt := time.Now().Add(tt.wantDelay)
				diff := entry.FireAt().Sub(wantFireAt)
				if diff < -2*time.Second || diff > 2*time.Second {
					t.Fatalf("FireAt = %v, want approximately %v (diff=%v)", entry.FireAt(), wantFireAt, diff)
				}
			}
		})
	}
}

func TestSelfDestructTimerTaskFiresOnElapsedEntry(t *testing.T) {
	r := GetMonsterRegistry()
	r.Clear(context.Background())
	ten := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), ten)

	capture := producertest.InstallCapturing()
	t.Cleanup(producertest.InstallNoop)

	prevHook := testInformationLookup
	testInformationLookup = func(_ uint32) (information.Model, error) {
		return information.NewModelBuilder().SetSelfDestruction(timerSelfDestruction()).Build(), nil
	}
	t.Cleanup(func() { testInformationLookup = prevHook })

	m := r.CreateMonster(ctx, ten, testField(), timerMobMonsterId, 0, 0, 0, 5, 0, timerMobMaxHp, 0, "", "")
	uid := m.UniqueId()
	now := time.Now()
	entry := NewSelfDestructTimerEntry(timerMobMonsterId, testField(), 4, now.Add(-time.Second))
	GetSelfDestructTimerRegistry().Register(ctx, ten, uid, entry)

	task := &SelfDestructTimerTask{l: logrus.New(), ctx: context.Background()}
	task.processEntry(now, ten, uid, entry)

	bodies := killedEvents(t, capture)
	require.Len(t, bodies, 1)
	require.Equal(t, byte(4), bodies[0].DeathType)
	require.Equal(t, uint32(0), bodies[0].ActorId)

	_, err := r.GetMonster(ten, uid)
	require.Error(t, err, "monster must be absent from the registry after detonation")

	entries := entriesForTenant(ctx, ten)
	require.Empty(t, entries)
}

func TestSelfDestructTimerTaskSkipsUnelapsedEntry(t *testing.T) {
	r := GetMonsterRegistry()
	r.Clear(context.Background())
	ten := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), ten)

	capture := producertest.InstallCapturing()
	t.Cleanup(producertest.InstallNoop)

	prevHook := testInformationLookup
	testInformationLookup = func(_ uint32) (information.Model, error) {
		return information.NewModelBuilder().SetSelfDestruction(timerSelfDestruction()).Build(), nil
	}
	t.Cleanup(func() { testInformationLookup = prevHook })

	m := r.CreateMonster(ctx, ten, testField(), timerMobMonsterId, 0, 0, 0, 5, 0, timerMobMaxHp, 0, "", "")
	uid := m.UniqueId()
	now := time.Now()
	entry := NewSelfDestructTimerEntry(timerMobMonsterId, testField(), 4, now.Add(time.Minute))
	GetSelfDestructTimerRegistry().Register(ctx, ten, uid, entry)
	t.Cleanup(func() { GetSelfDestructTimerRegistry().Unregister(ctx, ten, uid) })

	task := &SelfDestructTimerTask{l: logrus.New(), ctx: context.Background()}
	task.processEntry(now, ten, uid, entry)

	require.Empty(t, killedEvents(t, capture))

	_, err := r.GetMonster(ten, uid)
	require.NoError(t, err, "monster must still be present")

	entries := entriesForTenant(ctx, ten)
	require.Len(t, entries, 1, "unelapsed entry must remain armed")
}

func TestSelfDestructTimerTaskUnregistersDeadMob(t *testing.T) {
	r := GetMonsterRegistry()
	r.Clear(context.Background())
	ten := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), ten)

	capture := producertest.InstallCapturing()
	t.Cleanup(producertest.InstallNoop)

	prevHook := testInformationLookup
	testInformationLookup = func(_ uint32) (information.Model, error) {
		return information.NewModelBuilder().SetSelfDestruction(timerSelfDestruction()).Build(), nil
	}
	t.Cleanup(func() { testInformationLookup = prevHook })

	m := r.CreateMonster(ctx, ten, testField(), timerMobMonsterId, 0, 0, 0, 5, 0, timerMobMaxHp, 0, "", "")
	uid := m.UniqueId()
	now := time.Now()
	entry := NewSelfDestructTimerEntry(timerMobMonsterId, testField(), 4, now.Add(-time.Second))
	GetSelfDestructTimerRegistry().Register(ctx, ten, uid, entry)

	_, err := r.SelfDestruct(ten, uid)
	require.NoError(t, err)

	task := &SelfDestructTimerTask{l: logrus.New(), ctx: context.Background()}
	task.processEntry(now, ten, uid, entry)

	require.Empty(t, killedEvents(t, capture), "the dead mob must not produce a second KILLED")

	entries := entriesForTenant(ctx, ten)
	require.Empty(t, entries)
}

func TestSelfDestructTimerTaskUnregistersMissingMob(t *testing.T) {
	r := GetMonsterRegistry()
	r.Clear(context.Background())
	ten := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), ten)

	capture := producertest.InstallCapturing()
	t.Cleanup(producertest.InstallNoop)

	uid := uint32(424242)
	now := time.Now()
	entry := NewSelfDestructTimerEntry(timerMobMonsterId, testField(), 4, now.Add(-time.Second))
	GetSelfDestructTimerRegistry().Register(ctx, ten, uid, entry)

	task := &SelfDestructTimerTask{l: logrus.New(), ctx: context.Background()}
	task.processEntry(now, ten, uid, entry)

	require.Empty(t, killedEvents(t, capture))

	entries := entriesForTenant(ctx, ten)
	require.Empty(t, entries)
}

func TestKillUnregistersTimer(t *testing.T) {
	r := GetMonsterRegistry()
	r.Clear(context.Background())
	ten := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), ten)

	prevHook := testInformationLookup
	testInformationLookup = func(_ uint32) (information.Model, error) {
		return information.NewModelBuilder().SetSelfDestruction(timerSelfDestruction()).Build(), nil
	}
	t.Cleanup(func() { testInformationLookup = prevHook })

	m := r.CreateMonster(ctx, ten, testField(), timerMobMonsterId, 0, 0, 0, 5, 0, timerMobMaxHp, 0, "", "")
	uid := m.UniqueId()
	entry := NewSelfDestructTimerEntry(timerMobMonsterId, testField(), 4, time.Now().Add(time.Minute))
	GetSelfDestructTimerRegistry().Register(ctx, ten, uid, entry)

	p, _ := newRecordingProcessorWithBodies(t, ten)
	p.ctx = ctx
	p.damageCore(m, 55, []uint32{timerMobMaxHp})

	entries := entriesForTenant(ctx, ten)
	require.Empty(t, entries, "FR-3.3: an ordinary kill must cancel the armed timer")
}

func TestDestroyUnregistersTimer(t *testing.T) {
	r := GetMonsterRegistry()
	r.Clear(context.Background())
	ten := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), ten)

	m := r.CreateMonster(ctx, ten, testField(), timerMobMonsterId, 0, 0, 0, 5, 0, timerMobMaxHp, 0, "", "")
	uid := m.UniqueId()
	entry := NewSelfDestructTimerEntry(timerMobMonsterId, testField(), 4, time.Now().Add(time.Minute))
	GetSelfDestructTimerRegistry().Register(ctx, ten, uid, entry)

	p := NewProcessor(logrus.New(), ctx)
	require.NoError(t, p.Destroy(uid))

	entries := entriesForTenant(ctx, ten)
	require.Empty(t, entries)
}

func TestDestroyAllLeavesNoTimers(t *testing.T) {
	r := GetMonsterRegistry()
	r.Clear(context.Background())
	ten1 := newTestTenant(t)
	ten2 := newTestTenant(t)

	prevHook := testInformationLookup
	testInformationLookup = func(_ uint32) (information.Model, error) {
		return information.NewModelBuilder().SetSelfDestruction(timerSelfDestruction()).Build(), nil
	}
	t.Cleanup(func() { testInformationLookup = prevHook })

	ctx1 := tenant.WithContext(context.Background(), ten1)
	ctx2 := tenant.WithContext(context.Background(), ten2)

	_, err := NewProcessor(logrus.New(), ctx1).Create(testField(), RestModel{MonsterId: timerMobMonsterId})
	require.NoError(t, err)
	_, err = NewProcessor(logrus.New(), ctx2).Create(testField(), RestModel{MonsterId: timerMobMonsterId})
	require.NoError(t, err)

	require.Len(t, entriesForTenant(ctx1, ten1), 1)
	require.Len(t, entriesForTenant(ctx2, ten2), 1)

	require.NoError(t, DestroyAll(logrus.New(), context.Background()))

	require.Empty(t, entriesForTenant(ctx1, ten1))
	require.Empty(t, entriesForTenant(ctx2, ten2))
}
