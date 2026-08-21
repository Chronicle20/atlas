package monster

import (
	"atlas-monsters/monster/information"
	"context"
	"errors"
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// setAggroProcessor is forceControlProcessor plus a fixed clock, so
// SetAggro's lease-stamp assertions are deterministic.
func setAggroProcessor(ctx context.Context, tm tenant.Model, emitted *int, inField []uint32, hidden map[uint32]struct{}, nowMs int64) *ProcessorImpl {
	p := forceControlProcessor(ctx, tm, emitted, inField, hidden)
	p.nowFn = func() int64 { return nowMs }
	return p
}

func TestSetAggro_Gates(t *testing.T) {
	type tc struct {
		name            string
		seedController  uint32
		killMonster     bool
		infoHook        func(_ uint32) (information.Model, error)
		claimant        uint32
		inField         []uint32
		wantEmitted     int
		checkGetById    bool
		wantHasAggro    bool
		wantControlChar uint32
	}

	tests := []tc{
		{
			name: "unknown monster",
			infoHook: func(_ uint32) (information.Model, error) {
				return information.NewModelBuilder().SetFirstAttack(true).Build(), nil
			},
			claimant:     9,
			inField:      []uint32{9},
			wantEmitted:  0,
			checkGetById: false,
		},
		{
			name:           "dead monster",
			seedController: 7,
			killMonster:    true,
			infoHook: func(_ uint32) (information.Model, error) {
				return information.NewModelBuilder().SetFirstAttack(true).Build(), nil
			},
			claimant:        7,
			inField:         []uint32{7},
			wantEmitted:     0,
			checkGetById:    true,
			wantHasAggro:    true, // ApplyDamage already flipped this true on the killing blow
			wantControlChar: 7,
		},
		{
			name:           "passive template",
			seedController: 7,
			infoHook: func(_ uint32) (information.Model, error) {
				return information.NewModelBuilder().SetFirstAttack(false).Build(), nil
			},
			claimant:        7,
			inField:         []uint32{7},
			wantEmitted:     0,
			checkGetById:    true,
			wantHasAggro:    false,
			wantControlChar: 7,
		},
		{
			name:            "information lookup error",
			seedController:  7,
			infoHook:        func(_ uint32) (information.Model, error) { return information.Model{}, errors.New("boom") },
			claimant:        7,
			inField:         []uint32{7},
			wantEmitted:     0,
			checkGetById:    true,
			wantHasAggro:    false,
			wantControlChar: 7,
		},
		{
			name:           "character not in field",
			seedController: 7,
			infoHook: func(_ uint32) (information.Model, error) {
				return information.NewModelBuilder().SetFirstAttack(true).Build(), nil
			},
			claimant:        9,
			inField:         []uint32{7},
			wantEmitted:     0,
			checkGetById:    true,
			wantHasAggro:    false,
			wantControlChar: 7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm := newTestTenant(t)
			ctx := tenant.WithContext(context.Background(), tm)
			GetMonsterRegistry().Clear(ctx)

			prevHook := testInformationLookup
			testInformationLookup = tt.infoHook
			defer func() { testInformationLookup = prevHook }()

			var uid uint32
			if tt.name != "unknown monster" {
				uid = newAggroedMonster(t, ctx, tm, tt.seedController, nil)
				if tt.killMonster {
					if _, err := GetMonsterRegistry().ApplyDamage(tm, tt.seedController, 100000, uid, 1000); err != nil {
						t.Fatalf("ApplyDamage: %v", err)
					}
				}
			} else {
				uid = 4242
			}

			emitted := 0
			p := setAggroProcessor(ctx, tm, &emitted, tt.inField, map[uint32]struct{}{}, 5000)

			if err := p.SetAggro(uid, tt.claimant); err != nil {
				t.Fatalf("SetAggro: %v", err)
			}
			if emitted != tt.wantEmitted {
				t.Fatalf("emitted %d events, want %d", emitted, tt.wantEmitted)
			}

			if tt.checkGetById {
				got, err := p.GetById(uid)
				if err != nil {
					t.Fatalf("GetById: %v", err)
				}
				if got.ControllerHasAggro() != tt.wantHasAggro {
					t.Fatalf("ControllerHasAggro = %v, want %v", got.ControllerHasAggro(), tt.wantHasAggro)
				}
				if got.ControlCharacterId() != tt.wantControlChar {
					t.Fatalf("ControlCharacterId = %d, want %d", got.ControlCharacterId(), tt.wantControlChar)
				}
			}
		})
	}
}

func TestSetAggro_Arbitration(t *testing.T) {
	const nowMs = int64(5000)

	prevHook := testInformationLookup
	testInformationLookup = func(_ uint32) (information.Model, error) {
		return information.NewModelBuilder().SetFirstAttack(true).Build(), nil
	}
	defer func() { testInformationLookup = prevHook }()

	t.Run("controller without aggro flips and emits once", func(t *testing.T) {
		tm := newTestTenant(t)
		ctx := tenant.WithContext(context.Background(), tm)
		GetMonsterRegistry().Clear(ctx)

		uid := newAggroedMonster(t, ctx, tm, 0, nil)
		if _, err := GetMonsterRegistry().ControlMonster(tm, uid, 7); err != nil {
			t.Fatalf("ControlMonster: %v", err)
		}

		emitted := 0
		p := setAggroProcessor(ctx, tm, &emitted, []uint32{7, 9}, map[uint32]struct{}{}, nowMs)

		if err := p.SetAggro(uid, 7); err != nil {
			t.Fatalf("SetAggro: %v", err)
		}
		if emitted != 1 {
			t.Fatalf("emitted %d events, want 1", emitted)
		}
		got, err := p.GetById(uid)
		if err != nil {
			t.Fatalf("GetById: %v", err)
		}
		if !got.ControllerHasAggro() {
			t.Fatal("ControllerHasAggro = false, want true")
		}
		if got.ControlCharacterId() != 7 {
			t.Fatalf("ControlCharacterId = %d, want 7", got.ControlCharacterId())
		}
		if got.AggroRefreshedMs() != nowMs {
			t.Fatalf("AggroRefreshedMs = %d, want %d", got.AggroRefreshedMs(), nowMs)
		}
	})

	t.Run("controller with aggro stamps lease, emits nothing", func(t *testing.T) {
		tm := newTestTenant(t)
		ctx := tenant.WithContext(context.Background(), tm)
		GetMonsterRegistry().Clear(ctx)

		uid := newAggroedMonster(t, ctx, tm, 0, nil)
		if _, err := GetMonsterRegistry().ControlMonsterWithAggro(tm, uid, 7, 1000); err != nil {
			t.Fatalf("ControlMonsterWithAggro: %v", err)
		}

		emitted := 0
		p := setAggroProcessor(ctx, tm, &emitted, []uint32{7, 9}, map[uint32]struct{}{}, nowMs)

		if err := p.SetAggro(uid, 7); err != nil {
			t.Fatalf("SetAggro: %v", err)
		}
		if emitted != 0 {
			t.Fatalf("emitted %d events, want 0", emitted)
		}
		got, err := p.GetById(uid)
		if err != nil {
			t.Fatalf("GetById: %v", err)
		}
		if !got.ControllerHasAggro() {
			t.Fatal("ControllerHasAggro = false, want true")
		}
		if got.ControlCharacterId() != 7 {
			t.Fatalf("ControlCharacterId = %d, want 7", got.ControlCharacterId())
		}
		if got.AggroRefreshedMs() != nowMs {
			t.Fatalf("AggroRefreshedMs = %d, want %d", got.AggroRefreshedMs(), nowMs)
		}
	})

	t.Run("non-controller takes control with aggro", func(t *testing.T) {
		tm := newTestTenant(t)
		ctx := tenant.WithContext(context.Background(), tm)
		GetMonsterRegistry().Clear(ctx)

		uid := newAggroedMonster(t, ctx, tm, 0, nil)
		if _, err := GetMonsterRegistry().ControlMonster(tm, uid, 7); err != nil {
			t.Fatalf("ControlMonster: %v", err)
		}

		emitted := 0
		p := setAggroProcessor(ctx, tm, &emitted, []uint32{7, 9}, map[uint32]struct{}{}, nowMs)

		if err := p.SetAggro(uid, 9); err != nil {
			t.Fatalf("SetAggro: %v", err)
		}
		// STOP_CONTROL + START_CONTROL, plus RepickAndEmit's unconditional
		// NEXT_SKILL_DECIDED once ControllerHasAggro flips true (same
		// startControl(forceAggro=true) path ForceControl uses).
		if emitted != 3 {
			t.Fatalf("emitted %d events, want 3 (stop, start, next-skill)", emitted)
		}
		got, err := p.GetById(uid)
		if err != nil {
			t.Fatalf("GetById: %v", err)
		}
		if !got.ControllerHasAggro() {
			t.Fatal("ControllerHasAggro = false, want true")
		}
		if got.ControlCharacterId() != 9 {
			t.Fatalf("ControlCharacterId = %d, want 9", got.ControlCharacterId())
		}
		if got.AggroRefreshedMs() != nowMs {
			t.Fatalf("AggroRefreshedMs = %d, want %d", got.AggroRefreshedMs(), nowMs)
		}
	})

	t.Run("non-controller loses to an existing aggro holder", func(t *testing.T) {
		tm := newTestTenant(t)
		ctx := tenant.WithContext(context.Background(), tm)
		GetMonsterRegistry().Clear(ctx)

		uid := newAggroedMonster(t, ctx, tm, 0, nil)
		if _, err := GetMonsterRegistry().ControlMonsterWithAggro(tm, uid, 7, 1000); err != nil {
			t.Fatalf("ControlMonsterWithAggro: %v", err)
		}

		emitted := 0
		p := setAggroProcessor(ctx, tm, &emitted, []uint32{7, 9}, map[uint32]struct{}{}, nowMs)

		if err := p.SetAggro(uid, 9); err != nil {
			t.Fatalf("SetAggro: %v", err)
		}
		if emitted != 0 {
			t.Fatalf("emitted %d events, want 0", emitted)
		}
		got, err := p.GetById(uid)
		if err != nil {
			t.Fatalf("GetById: %v", err)
		}
		if !got.ControllerHasAggro() {
			t.Fatal("ControllerHasAggro = false, want true")
		}
		if got.ControlCharacterId() != 7 {
			t.Fatalf("ControlCharacterId = %d, want 7", got.ControlCharacterId())
		}
		if got.AggroRefreshedMs() != 1000 {
			t.Fatalf("AggroRefreshedMs = %d, want 1000 (unchanged)", got.AggroRefreshedMs())
		}
	})

	t.Run("uncontrolled monster is claimed", func(t *testing.T) {
		tm := newTestTenant(t)
		ctx := tenant.WithContext(context.Background(), tm)
		GetMonsterRegistry().Clear(ctx)

		uid := newAggroedMonster(t, ctx, tm, 0, nil)

		emitted := 0
		p := setAggroProcessor(ctx, tm, &emitted, []uint32{7, 9}, map[uint32]struct{}{}, nowMs)

		if err := p.SetAggro(uid, 9); err != nil {
			t.Fatalf("SetAggro: %v", err)
		}
		// START_CONTROL, plus RepickAndEmit's unconditional NEXT_SKILL_DECIDED
		// once ControllerHasAggro flips true — see the note on the
		// "non-controller takes control with aggro" case above.
		if emitted != 2 {
			t.Fatalf("emitted %d events, want 2 (start, next-skill)", emitted)
		}
		got, err := p.GetById(uid)
		if err != nil {
			t.Fatalf("GetById: %v", err)
		}
		if !got.ControllerHasAggro() {
			t.Fatal("ControllerHasAggro = false, want true")
		}
		if got.ControlCharacterId() != 9 {
			t.Fatalf("ControlCharacterId = %d, want 9", got.ControlCharacterId())
		}
		if got.AggroRefreshedMs() != nowMs {
			t.Fatalf("AggroRefreshedMs = %d, want %d", got.AggroRefreshedMs(), nowMs)
		}
	})

	t.Run("GM-hidden claimant is dropped", func(t *testing.T) {
		tm := newTestTenant(t)
		ctx := tenant.WithContext(context.Background(), tm)
		GetMonsterRegistry().Clear(ctx)

		uid := newAggroedMonster(t, ctx, tm, 0, nil)
		if _, err := GetMonsterRegistry().ControlMonster(tm, uid, 7); err != nil {
			t.Fatalf("ControlMonster: %v", err)
		}

		emitted := 0
		p := setAggroProcessor(ctx, tm, &emitted, []uint32{7, 9}, map[uint32]struct{}{9: {}}, nowMs)

		if err := p.SetAggro(uid, 9); err != nil {
			t.Fatalf("SetAggro: %v", err)
		}
		if emitted != 0 {
			t.Fatalf("emitted %d events, want 0", emitted)
		}
		got, err := p.GetById(uid)
		if err != nil {
			t.Fatalf("GetById: %v", err)
		}
		if got.ControllerHasAggro() {
			t.Fatal("ControllerHasAggro = true, want false")
		}
		if got.ControlCharacterId() != 7 {
			t.Fatalf("ControlCharacterId = %d, want 7", got.ControlCharacterId())
		}
		if got.AggroRefreshedMs() != 0 {
			t.Fatalf("AggroRefreshedMs = %d, want 0", got.AggroRefreshedMs())
		}
	})
}

// TestSetAggro_LeavesDamageEntriesUntouched guards FR-4.5: auto-aggro confers
// no drop ownership and no kill credit, so SetAggro must never write a damage
// entry.
func TestSetAggro_LeavesDamageEntriesUntouched(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	GetMonsterRegistry().Clear(ctx)

	prevHook := testInformationLookup
	testInformationLookup = func(_ uint32) (information.Model, error) {
		return information.NewModelBuilder().SetFirstAttack(true).Build(), nil
	}
	defer func() { testInformationLookup = prevHook }()

	uid := newAggroedMonster(t, ctx, tm, 7, []uint32{8})

	emitted := 0
	p := setAggroProcessor(ctx, tm, &emitted, []uint32{7, 8}, map[uint32]struct{}{}, 5000)

	if err := p.SetAggro(uid, 7); err != nil {
		t.Fatalf("SetAggro: %v", err)
	}

	got, err := p.GetById(uid)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if len(got.DamageEntries()) != 1 {
		t.Fatalf("DamageEntries len = %d, want 1", len(got.DamageEntries()))
	}
	if got.DamageEntries()[0].CharacterId != 8 {
		t.Fatalf("DamageEntries[0].CharacterId = %d, want 8", got.DamageEntries()[0].CharacterId)
	}
}
