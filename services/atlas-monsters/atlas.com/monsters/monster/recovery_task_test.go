package monster

import (
	"atlas-monsters/monster/information"
	"context"
	"testing"
	"time"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func TestRecoveryTask_AppliesMpAndEmitsHp(t *testing.T) {
	r := GetMonsterRegistry()
	ctx := context.Background()
	r.Clear(ctx)

	tm := newTestTenant(t)
	tctx := tenant.WithContext(ctx, tm)

	m := r.CreateMonster(tctx, tm, testField(), 9300018, 0, 0, 0, 5, 0, 1000, 100)
	if _, err := r.ControlMonster(tm, m.UniqueId(), 99); err != nil {
		t.Fatalf("ControlMonster: %v", err)
	}
	dmgAt := time.Now().Add(-30 * time.Second).UnixMilli()
	if _, err := r.ApplyDamage(tm, 99, 200, m.UniqueId(), dmgAt); err != nil {
		t.Fatalf("ApplyDamage: %v", err)
	}
	if _, err := r.DeductMp(tm, m.UniqueId(), 50); err != nil {
		t.Fatalf("DeductMp: %v", err)
	}

	emits := 0
	tk := &MonsterRecoveryTask{
		l:        newPickerLogger(),
		ctx:      ctx,
		interval: MonsterRecoveryInterval,
		nowFn:    func() int64 { return time.Now().UnixMilli() },
		infoFn: func(_ tenant.Model, _ uint32) (information.Model, error) {
			return information.NewModelBuilder().
				SetHpRecovery(50).SetMpRecovery(5).Build(), nil
		},
		applyFn: r.ApplyRecovery,
		emitFn: func(_ tenant.Model, _ Model) error {
			emits++
			return nil
		},
	}
	tk.Run()

	got, err := r.GetMonster(tm, m.UniqueId())
	if err != nil {
		t.Fatalf("GetMonster: %v", err)
	}
	if got.Mp() != 55 {
		t.Errorf("MP after recovery: got %d, want 55", got.Mp())
	}
	if got.Hp() != 850 {
		t.Errorf("HP after recovery: got %d, want 850 (was 800 + 50 regen)", got.Hp())
	}
	if emits != 1 {
		t.Errorf("expected 1 HP-bar emit; got %d", emits)
	}
}

func TestRecoveryTask_SkipsBothZero(t *testing.T) {
	r := GetMonsterRegistry()
	ctx := context.Background()
	r.Clear(ctx)

	tm := newTestTenant(t)
	tctx := tenant.WithContext(ctx, tm)

	m := r.CreateMonster(tctx, tm, testField(), 9300018, 0, 0, 0, 5, 0, 1000, 100)
	if _, err := r.DeductMp(tm, m.UniqueId(), 50); err != nil {
		t.Fatalf("DeductMp: %v", err)
	}

	applyCalls := 0
	tk := &MonsterRecoveryTask{
		l:        newPickerLogger(),
		ctx:      ctx,
		interval: MonsterRecoveryInterval,
		nowFn:    func() int64 { return time.Now().UnixMilli() },
		infoFn: func(_ tenant.Model, _ uint32) (information.Model, error) {
			return information.NewModelBuilder().Build(), nil // both recoveries 0
		},
		applyFn: func(_ tenant.Model, _ uint32, _, _ uint32, _ int64) (Model, bool, bool, error) {
			applyCalls++
			return Model{}, false, false, nil
		},
		emitFn: func(_ tenant.Model, _ Model) error { return nil },
	}
	tk.Run()

	if applyCalls != 0 {
		t.Errorf("expected zero applyFn calls when both recoveries are 0; got %d", applyCalls)
	}
}

func TestRecoveryTask_SkipsFullHpAndFullMp(t *testing.T) {
	r := GetMonsterRegistry()
	ctx := context.Background()
	r.Clear(ctx)

	tm := newTestTenant(t)
	tctx := tenant.WithContext(ctx, tm)

	_ = r.CreateMonster(tctx, tm, testField(), 9300018, 0, 0, 0, 5, 0, 1000, 100)

	infoCalls := 0
	tk := &MonsterRecoveryTask{
		l:        newPickerLogger(),
		ctx:      ctx,
		interval: MonsterRecoveryInterval,
		nowFn:    func() int64 { return time.Now().UnixMilli() },
		infoFn: func(_ tenant.Model, _ uint32) (information.Model, error) {
			infoCalls++
			return information.NewModelBuilder().SetHpRecovery(10).SetMpRecovery(10).Build(), nil
		},
		applyFn: r.ApplyRecovery,
		emitFn:  func(_ tenant.Model, _ Model) error { return nil },
	}
	tk.Run()

	if infoCalls != 0 {
		t.Errorf("expected zero info lookups for at-cap mob; got %d", infoCalls)
	}
}

func TestRecoveryTask_SkipsDeadMob(t *testing.T) {
	r := GetMonsterRegistry()
	ctx := context.Background()
	r.Clear(ctx)

	tm := newTestTenant(t)
	tctx := tenant.WithContext(ctx, tm)

	m := r.CreateMonster(tctx, tm, testField(), 9300018, 0, 0, 0, 5, 0, 1, 100)
	if _, err := r.ApplyDamage(tm, 99, 1, m.UniqueId(), time.Now().UnixMilli()); err != nil {
		t.Fatalf("ApplyDamage: %v", err)
	}

	infoCalls := 0
	tk := &MonsterRecoveryTask{
		l:        newPickerLogger(),
		ctx:      ctx,
		interval: MonsterRecoveryInterval,
		nowFn:    func() int64 { return time.Now().UnixMilli() },
		infoFn: func(_ tenant.Model, _ uint32) (information.Model, error) {
			infoCalls++
			return information.NewModelBuilder().SetHpRecovery(50).SetMpRecovery(5).Build(), nil
		},
		applyFn: r.ApplyRecovery,
		emitFn:  func(_ tenant.Model, _ Model) error { return nil },
	}
	tk.Run()

	if infoCalls != 0 {
		t.Errorf("expected zero info lookups for dead mob; got %d", infoCalls)
	}
}
