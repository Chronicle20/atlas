package monster

import (
	"context"
	"testing"
	"time"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func TestPickerSweep_RepicksOnlyEligibleMonsters(t *testing.T) {
	r := GetMonsterRegistry()
	ctx := context.Background()
	r.Clear(ctx)

	tm := newTestTenant(t)
	tctx := tenant.WithContext(ctx, tm)

	// Monster A: nextEligibleRepickAtMs in the past — should be repicked.
	a := r.CreateMonster(tctx, tm, testField(), 9000000, 0, 0, 0, 0, 0, 100, 50)
	_, _ = r.SetNextSkillDecision(tm, a.UniqueId(), nextSkillDecision{
		nextEligibleRepickAtMs: time.Now().Add(-time.Second).UnixMilli(),
	})
	if _, err := r.ControlMonster(tm, a.UniqueId(), 99); err != nil {
		t.Fatalf("ControlMonster: %v", err)
	}
	if _, err := r.ApplyDamage(tm, 99, 1, a.UniqueId(), time.Now().UnixMilli()); err != nil {
		t.Fatalf("ApplyDamage: %v", err)
	}
	// Re-set decision because ApplyDamage doesn't touch nextSkillDecision.
	_, _ = r.SetNextSkillDecision(tm, a.UniqueId(), nextSkillDecision{
		nextEligibleRepickAtMs: time.Now().Add(-time.Second).UnixMilli(),
	})

	// Monster B: nextEligibleRepickAtMs sentinel zero — should be skipped.
	_ = r.CreateMonster(tctx, tm, testField(), 9000000, 1, 1, 0, 0, 0, 100, 50)

	// Monster C: nextEligibleRepickAtMs in the future — should be skipped.
	c := r.CreateMonster(tctx, tm, testField(), 9000000, 2, 2, 0, 0, 0, 100, 50)
	_, _ = r.SetNextSkillDecision(tm, c.UniqueId(), nextSkillDecision{
		nextEligibleRepickAtMs: time.Now().Add(time.Hour).UnixMilli(),
	})

	repicked := map[uint32]int{}
	tk := &MonsterSkillPickerSweepTask{
		l:        newPickerLogger(),
		ctx:      ctx,
		interval: 1500 * time.Millisecond,
		nowFn:    func() int64 { return time.Now().UnixMilli() },
		repickFn: func(t tenant.Model, uniqueId uint32) error {
			repicked[uniqueId]++
			return nil
		},
		hasSkillsFn: func(_ tenant.Model, monsterId uint32) bool { return true },
	}
	tk.Run()

	if repicked[a.UniqueId()] != 1 {
		t.Fatalf("expected monster A to be repicked once; got %d", repicked[a.UniqueId()])
	}
	// Monsters B and C should not appear in the repicked map.
	for uid, count := range repicked {
		if uid == a.UniqueId() {
			continue
		}
		if count != 0 {
			t.Fatalf("expected monster [%d] to be skipped; got %d repicks", uid, count)
		}
	}

	// Sanity: SleepTime compiles.
	var _ time.Duration = tk.SleepTime()
}

func TestPickerSweep_SkipsMonstersWithNoSkills(t *testing.T) {
	r := GetMonsterRegistry()
	ctx := context.Background()
	r.Clear(ctx)

	tm := newTestTenant(t)
	tctx := tenant.WithContext(ctx, tm)

	a := r.CreateMonster(tctx, tm, testField(), 9000000, 0, 0, 0, 0, 0, 100, 50)
	_, _ = r.SetNextSkillDecision(tm, a.UniqueId(), nextSkillDecision{
		nextEligibleRepickAtMs: time.Now().Add(-time.Second).UnixMilli(),
	})

	repicked := 0
	tk := &MonsterSkillPickerSweepTask{
		l:           newPickerLogger(),
		ctx:         ctx,
		interval:    1500 * time.Millisecond,
		nowFn:       func() int64 { return time.Now().UnixMilli() },
		repickFn:    func(_ tenant.Model, _ uint32) error { repicked++; return nil },
		hasSkillsFn: func(_ tenant.Model, _ uint32) bool { return false }, // no skills
	}
	tk.Run()

	if repicked != 0 {
		t.Fatalf("expected zero repicks for skill-less monster; got %d", repicked)
	}
}

func TestPickerSweep_SkipsWhenAggroFalse(t *testing.T) {
	r := GetMonsterRegistry()
	ctx := context.Background()
	r.Clear(ctx)

	tm := newTestTenant(t)
	tctx := tenant.WithContext(ctx, tm)

	a := r.CreateMonster(tctx, tm, testField(), 9000000, 0, 0, 0, 0, 0, 100, 50)
	_, _ = r.SetNextSkillDecision(tm, a.UniqueId(), nextSkillDecision{
		nextEligibleRepickAtMs: time.Now().Add(-time.Second).UnixMilli(),
	})
	// controllerHasAggro stays false (default after Create + no damage).

	repicked := 0
	tk := &MonsterSkillPickerSweepTask{
		l:           newPickerLogger(),
		ctx:         ctx,
		interval:    1500 * time.Millisecond,
		nowFn:       func() int64 { return time.Now().UnixMilli() },
		repickFn:    func(_ tenant.Model, _ uint32) error { repicked++; return nil },
		hasSkillsFn: func(_ tenant.Model, _ uint32) bool { return true },
	}
	tk.Run()

	if repicked != 0 {
		t.Fatalf("expected zero repicks for non-aggro monster; got %d", repicked)
	}
}

func TestPickerSweep_RepicksWhenAggroTrue(t *testing.T) {
	r := GetMonsterRegistry()
	ctx := context.Background()
	r.Clear(ctx)

	tm := newTestTenant(t)
	tctx := tenant.WithContext(ctx, tm)

	a := r.CreateMonster(tctx, tm, testField(), 9000000, 0, 0, 0, 0, 0, 100, 50)
	if _, err := r.ControlMonster(tm, a.UniqueId(), 99); err != nil {
		t.Fatalf("ControlMonster: %v", err)
	}
	if _, err := r.ApplyDamage(tm, 99, 1, a.UniqueId(), time.Now().UnixMilli()); err != nil {
		t.Fatalf("ApplyDamage: %v", err)
	}
	_, _ = r.SetNextSkillDecision(tm, a.UniqueId(), nextSkillDecision{
		nextEligibleRepickAtMs: time.Now().Add(-time.Second).UnixMilli(),
	})

	repicked := 0
	tk := &MonsterSkillPickerSweepTask{
		l:           newPickerLogger(),
		ctx:         ctx,
		interval:    1500 * time.Millisecond,
		nowFn:       func() int64 { return time.Now().UnixMilli() },
		repickFn:    func(_ tenant.Model, _ uint32) error { repicked++; return nil },
		hasSkillsFn: func(_ tenant.Model, _ uint32) bool { return true },
	}
	tk.Run()

	if repicked != 1 {
		t.Fatalf("expected 1 repick for aggro'd monster; got %d", repicked)
	}
}
