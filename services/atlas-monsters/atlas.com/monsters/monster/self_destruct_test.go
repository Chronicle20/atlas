package monster

import (
	"atlas-monsters/monster/information"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// boomerMonsterId and boomerMaxHp are the task-253 test fixture: monsterId
// 5100002, max HP 4000, selfDestruction {action: 1, removeAfter: -1, hp: 1800}.
const (
	boomerMonsterId = uint32(5100002)
	boomerMaxHp     = uint32(4000)
)

func boomerSelfDestruction() information.SelfDestruction {
	return information.NewSelfDestruction(true, 1, -1, 1800)
}

// countKilled counts EventMonsterStatusKilled messages among the recorded
// events and returns the decoded body of the last one (nil if none).
func countKilled(events []emittedBody) (int, *statusEventKilledBody) {
	count := 0
	var last *statusEventKilledBody
	for _, e := range events {
		if e.Type == EventMonsterStatusKilled {
			count++
			var body statusEventKilledBody
			if err := json.Unmarshal(e.Body, &body); err == nil {
				b := body
				last = &b
			}
		}
	}
	return count, last
}

func TestDamageCoreCrossingThresholdDetonates(t *testing.T) {
	tests := []struct {
		name           string
		startHp        uint32
		sd             information.SelfDestruction
		damages        []uint32
		wantKilled     int
		wantDeathType  string
		wantActorId    uint32
		checkDeathType bool
	}{
		{
			name:           "crosses threshold",
			startHp:        boomerMaxHp,
			sd:             boomerSelfDestruction(),
			damages:        []uint32{2300},
			wantKilled:     1,
			wantDeathType:  DeathTypeFadeOut,
			wantActorId:    55,
			checkDeathType: true,
		},
		{
			name:           "lands exactly on threshold",
			startHp:        boomerMaxHp,
			sd:             boomerSelfDestruction(),
			damages:        []uint32{2200},
			wantKilled:     1,
			wantDeathType:  DeathTypeFadeOut,
			wantActorId:    55,
			checkDeathType: true,
		},
		{
			name:       "stays above threshold",
			startHp:    boomerMaxHp,
			sd:         boomerSelfDestruction(),
			damages:    []uint32{100},
			wantKilled: 0,
		},
		{
			name:           "multi-line, crosses on line 2",
			startHp:        boomerMaxHp,
			sd:             boomerSelfDestruction(),
			damages:        []uint32{1500, 1500},
			wantKilled:     1,
			wantDeathType:  DeathTypeFadeOut,
			wantActorId:    55,
			checkDeathType: true,
		},
		{
			name:           "already below threshold, next hit",
			startHp:        1000,
			sd:             boomerSelfDestruction(),
			damages:        []uint32{1},
			wantKilled:     1,
			wantDeathType:  DeathTypeFadeOut,
			wantActorId:    55,
			checkDeathType: true,
		},
		{
			name:           "ordinary kill wins (damage reaches 0)",
			startHp:        boomerMaxHp,
			sd:             boomerSelfDestruction(),
			damages:        []uint32{4000},
			wantKilled:     1,
			wantDeathType:  DeathTypeFadeOut,
			wantActorId:    55,
			checkDeathType: true,
		},
		{
			name:           "no block - regression",
			startHp:        boomerMaxHp,
			sd:             information.NewSelfDestruction(false, 0, -1, -1),
			damages:        []uint32{4000},
			wantKilled:     1,
			wantDeathType:  DeathTypeFadeOut,
			wantActorId:    55,
			checkDeathType: true,
		},
		{
			name:       "no block, partial damage - regression",
			startHp:    boomerMaxHp,
			sd:         information.NewSelfDestruction(false, 0, -1, -1),
			damages:    []uint32{3999},
			wantKilled: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := GetMonsterRegistry()
			ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
			ctx := context.Background()
			r.Clear(ctx)

			prevHook := testInformationLookup
			testInformationLookup = func(_ uint32) (information.Model, error) {
				return information.NewModelBuilder().SetSelfDestruction(tt.sd).Build(), nil
			}
			defer func() { testInformationLookup = prevHook }()

			f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
			m := r.CreateMonster(ctx, ten, f, boomerMonsterId, 0, 0, 0, 5, 0, tt.startHp, 0, "", "")
			uid := m.UniqueId()

			p, events := newRecordingProcessorWithBodies(t, ten)
			p.damageCore(m, 55, tt.damages)

			gotKilled, body := countKilled(*events)
			if gotKilled != tt.wantKilled {
				t.Fatalf("KILLED count = %d, want %d: %v", gotKilled, tt.wantKilled, *events)
			}
			if tt.checkDeathType {
				if body == nil {
					t.Fatalf("expected a KILLED body, got none")
				}
				if body.DeathType != tt.wantDeathType {
					t.Errorf("DeathType = %s, want %s", body.DeathType, tt.wantDeathType)
				}
				if body.ActorId != tt.wantActorId {
					t.Errorf("ActorId = %d, want %d", body.ActorId, tt.wantActorId)
				}
			}

			_, err := GetMonsterRegistry().GetMonster(ten, uid)
			if gotKilled > 0 && err == nil {
				t.Errorf("expected monster [%d] absent from registry after a KILLED, but it is present", uid)
			}
			if gotKilled == 0 && err != nil {
				t.Errorf("expected monster [%d] present in registry when no KILLED was emitted, got error: %v", uid, err)
			}
		})
	}
}

func TestSelfDestructRejects(t *testing.T) {
	type setup func(t *testing.T, r *Registry, ten tenant.Model, f field.Model) uint32

	tests := []struct {
		name       string
		setup      setup
		hook       func(_ uint32) (information.Model, error)
		wantKilled int
	}{
		{
			name: "unknown monster",
			setup: func(t *testing.T, r *Registry, ten tenant.Model, f field.Model) uint32 {
				return 999999
			},
			hook: func(_ uint32) (information.Model, error) {
				return information.NewModelBuilder().SetSelfDestruction(boomerSelfDestruction()).Build(), nil
			},
			wantKilled: 0,
		},
		{
			name: "already dead",
			setup: func(t *testing.T, r *Registry, ten tenant.Model, f field.Model) uint32 {
				m := r.CreateMonster(context.Background(), ten, f, boomerMonsterId, 0, 0, 0, 5, 0, boomerMaxHp, 0, "", "")
				if _, err := r.SelfDestruct(ten, m.UniqueId()); err != nil {
					t.Fatalf("seed SelfDestruct: %v", err)
				}
				return m.UniqueId()
			},
			hook: func(_ uint32) (information.Model, error) {
				return information.NewModelBuilder().SetSelfDestruction(boomerSelfDestruction()).Build(), nil
			},
			wantKilled: 0,
		},
		{
			name: "no selfDestruction block",
			setup: func(t *testing.T, r *Registry, ten tenant.Model, f field.Model) uint32 {
				m := r.CreateMonster(context.Background(), ten, f, boomerMonsterId, 0, 0, 0, 5, 0, boomerMaxHp, 0, "", "")
				return m.UniqueId()
			},
			hook: func(_ uint32) (information.Model, error) {
				return information.NewModelBuilder().SetSelfDestruction(information.NewSelfDestruction(false, 0, -1, -1)).Build(), nil
			},
			wantKilled: 0,
		},
		{
			name: "information lookup fails",
			setup: func(t *testing.T, r *Registry, ten tenant.Model, f field.Model) uint32 {
				m := r.CreateMonster(context.Background(), ten, f, boomerMonsterId, 0, 0, 0, 5, 0, boomerMaxHp, 0, "", "")
				return m.UniqueId()
			},
			hook: func(_ uint32) (information.Model, error) {
				return information.Model{}, errors.New("atlas-data unavailable")
			},
			wantKilled: 0,
		},
		{
			name: "valid target",
			setup: func(t *testing.T, r *Registry, ten tenant.Model, f field.Model) uint32 {
				m := r.CreateMonster(context.Background(), ten, f, boomerMonsterId, 0, 0, 0, 5, 0, boomerMaxHp, 0, "", "")
				return m.UniqueId()
			},
			hook: func(_ uint32) (information.Model, error) {
				return information.NewModelBuilder().SetSelfDestruction(information.NewSelfDestruction(true, 3, -1, 5000)).Build(), nil
			},
			wantKilled: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := GetMonsterRegistry()
			ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
			ctx := context.Background()
			r.Clear(ctx)

			f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()

			prevHook := testInformationLookup
			testInformationLookup = tt.hook
			defer func() { testInformationLookup = prevHook }()

			uid := tt.setup(t, r, ten, f)

			p, events := newRecordingProcessorWithBodies(t, ten)
			p.SelfDestruct(uid, 0, TriggerContact)

			gotKilled, body := countKilled(*events)
			if gotKilled != tt.wantKilled {
				t.Fatalf("KILLED count = %d, want %d: %v", gotKilled, tt.wantKilled, *events)
			}
			if tt.wantKilled == 1 {
				if body == nil {
					t.Fatalf("expected a KILLED body")
				}
				// Contact detonations always resolve to DeathTypeBomb
				// regardless of the WZ action byte (task-253
				// bug-darkstar-no-explosion-or-damage.md): TryFirstSelfDestruction
				// (opcode 0xC1) sends TriggerContact, and the WZ action byte
				// (3, DESTRUCT_BY_MISS) would otherwise divert v83's
				// CMob::OnDie to a one-time action with no die1..dieN
				// fallback and no explosion.
				if body.DeathType != DeathTypeBomb {
					t.Errorf("DeathType = %s, want %s", body.DeathType, DeathTypeBomb)
				}
			}
		})
	}
}

// TestSelfDestructContactAlwaysBomb verifies TriggerContact detonates with
// DeathTypeBomb regardless of the WZ selfDestruction.action byte, while
// TriggerThreshold and TriggerTimer keep passing the WZ-derived deathType
// through unchanged (task-253 bug-darkstar-no-explosion-or-damage.md, Fix —
// DECIDED 2026-08-25).
func TestSelfDestructContactAlwaysBomb(t *testing.T) {
	tests := []struct {
		name      string
		trigger   SelfDestructTrigger
		action    byte
		wantDeath string
	}{
		{name: "contact with WZ action 1 (fadeOut) resolves to bomb", trigger: TriggerContact, action: 1, wantDeath: DeathTypeBomb},
		{name: "contact with WZ action 3 (destructByMiss) resolves to bomb", trigger: TriggerContact, action: 3, wantDeath: DeathTypeBomb},
		{name: "threshold keeps WZ-derived deathType", trigger: TriggerThreshold, action: 3, wantDeath: DeathTypeDestructByMiss},
		{name: "timer keeps WZ-derived deathType", trigger: TriggerTimer, action: 3, wantDeath: DeathTypeDestructByMiss},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := GetMonsterRegistry()
			ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
			ctx := context.Background()
			r.Clear(ctx)

			prevHook := testInformationLookup
			testInformationLookup = func(_ uint32) (information.Model, error) {
				return information.NewModelBuilder().SetSelfDestruction(information.NewSelfDestruction(true, tt.action, -1, 5000)).Build(), nil
			}
			defer func() { testInformationLookup = prevHook }()

			f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
			m := r.CreateMonster(ctx, ten, f, boomerMonsterId, 0, 0, 0, 5, 0, boomerMaxHp, 0, "", "")
			uid := m.UniqueId()

			p, events := newRecordingProcessorWithBodies(t, ten)
			p.SelfDestruct(uid, 0, tt.trigger)

			gotKilled, body := countKilled(*events)
			if gotKilled != 1 {
				t.Fatalf("KILLED count = %d, want 1: %v", gotKilled, *events)
			}
			if body.DeathType != tt.wantDeath {
				t.Errorf("DeathType = %s, want %s", body.DeathType, tt.wantDeath)
			}
		})
	}
}

func TestSelfDestructAttributesToDamageLeader(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	prevHook := testInformationLookup
	testInformationLookup = func(_ uint32) (information.Model, error) {
		return information.NewModelBuilder().SetSelfDestruction(boomerSelfDestruction()).Build(), nil
	}
	defer func() { testInformationLookup = prevHook }()

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, boomerMonsterId, 0, 0, 0, 5, 0, boomerMaxHp, 0, "", "")
	uid := m.UniqueId()

	if _, err := r.ApplyDamage(ten, 777, 100, uid, time.Now().UnixMilli()); err != nil {
		t.Fatalf("seed ApplyDamage: %v", err)
	}

	p, events := newRecordingProcessorWithBodies(t, ten)
	p.SelfDestruct(uid, 0, TriggerTimer)

	gotKilled, body := countKilled(*events)
	if gotKilled != 1 {
		t.Fatalf("KILLED count = %d, want 1: %v", gotKilled, *events)
	}
	if body.ActorId != 777 {
		t.Errorf("ActorId = %d, want 777 (damage leader)", body.ActorId)
	}
}

func TestSelfDestructNoDamageEntriesReportsNoKiller(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	prevHook := testInformationLookup
	testInformationLookup = func(_ uint32) (information.Model, error) {
		return information.NewModelBuilder().SetSelfDestruction(boomerSelfDestruction()).Build(), nil
	}
	defer func() { testInformationLookup = prevHook }()

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, boomerMonsterId, 0, 0, 0, 5, 0, boomerMaxHp, 0, "", "")
	uid := m.UniqueId()

	p, events := newRecordingProcessorWithBodies(t, ten)
	p.SelfDestruct(uid, 0, TriggerTimer)

	gotKilled, body := countKilled(*events)
	if gotKilled != 1 {
		t.Fatalf("KILLED count = %d, want 1: %v", gotKilled, *events)
	}
	if body.ActorId != 0 {
		t.Errorf("ActorId = %d, want 0 (no damage entries)", body.ActorId)
	}
}

func TestSelfDestructIsIdempotent(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)

	prevHook := testInformationLookup
	testInformationLookup = func(_ uint32) (information.Model, error) {
		return information.NewModelBuilder().SetSelfDestruction(boomerSelfDestruction()).Build(), nil
	}
	defer func() { testInformationLookup = prevHook }()

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, boomerMonsterId, 0, 0, 0, 5, 0, boomerMaxHp, 0, "", "")
	uid := m.UniqueId()

	p, events := newRecordingProcessorWithBodies(t, ten)
	p.SelfDestruct(uid, 0, TriggerContact)
	p.SelfDestruct(uid, 0, TriggerContact)

	gotKilled, _ := countKilled(*events)
	if gotKilled != 1 {
		t.Fatalf("KILLED count across two SelfDestruct calls = %d, want exactly 1: %v", gotKilled, *events)
	}
}
