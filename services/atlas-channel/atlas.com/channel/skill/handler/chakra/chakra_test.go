package chakra

import (
	"atlas-channel/character"
	"atlas-channel/data/skill/effect"
	"atlas-channel/effective_stats"
	"context"
	"errors"
	"testing"
	"time"

	chakrastate "atlas-channel/character/chakra"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"

	channelhandler "atlas-channel/skill/handler"
)

// TestHealDelta pins the end-to-end heal: base = 2.9 x effective LUK,
// scaled by the window's snapshotted y, clamped to missing HP.
func TestHealDelta(t *testing.T) {
	tests := []struct {
		name  string
		y     int16
		luck  uint32
		hp    uint16
		maxHp uint16
		want  int16
	}{
		{"v83 L1 y=68", 68, 100, 100, 1000, 197},
		{"v83 L30 y=300", 300, 100, 100, 1000, 870},
		{"v48 L1 y=9", 9, 100, 100, 1000, 26},
		{"v48 L30 y=200", 200, 100, 100, 1000, 580},
		{"v95 L10 y=300", 300, 100, 100, 1000, 870},
		{"clamped to missing hp", 300, 100, 950, 1000, 50},
		{"at full hp", 300, 100, 1000, 1000, 0},
		{"zero luck", 300, 0, 100, 1000, 0},
		{"zero y", 0, 100, 100, 1000, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := chakrastate.Entry{SkillLevel: 1, X: 99, Y: tc.y, StartedAt: time.Now()}
			if got := healDelta(e, tc.luck, tc.hp, tc.maxHp); got != tc.want {
				t.Fatalf("healDelta = %d, want %d", got, tc.want)
			}
		})
	}
}

// TestRegisteredOnIdentity pins PRD FR-9.1: the handler is installed on the
// version-blind identity, so one registration covers all eleven provisioned
// versions without a raw wire-id comparison anywhere.
func TestRegisteredOnIdentity(t *testing.T) {
	if _, ok := channelhandler.Lookup(skill2.ChiefBanditChakra); !ok {
		t.Fatal("no Handler registered for skill2.ChiefBanditChakra")
	}
}

// ---- Apply orchestration coverage -----------------------------------
//
// chakra.Apply's window-lookup/no-window path, the defer reg.Clear, the
// character-load-error path, the effective-stats failure fallback (base
// LUK / base MaxHp), the zero-delta skip, and the ChangeHP-error path are
// exercised below by overriding the package-level loadCaster /
// loadEffectiveStats / changeHP seams, mirroring
// skill/handler/mprecovery's loadCaster/changeHP/changeMP idiom. The
// recovery-window registry itself is the real process-wide singleton
// (chakrastate.GetRegistry()) -- each test uses its own tenant+character id
// so entries never collide, and every test that opens a window asserts it
// reads back absent afterward (the defer Clear fired exactly once).

func testLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetLevel(logrus.DebugLevel)
	return l
}

func testTenant(t *testing.T) tenant.Model {
	t.Helper()
	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tn
}

func testFieldModel() field.Model {
	return field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).Build()
}

func testInfo(level byte) packetmodel.SkillUsageInfo {
	return packetmodel.NewSkillUsageInfoBuilder().
		SetSkillId(uint32(skill2.ChiefBanditChakraId)).
		SetSkillLevel(level).
		Build()
}

// testEffectModel builds a zero-valued effect.Model through the REST
// extract path (no builder exists on the model; Apply never reads e's
// fields -- the recovery rate comes from the window snapshot, not the
// effect -- so its X/Y content is irrelevant to every case below).
func testEffectModel(t *testing.T) effect.Model {
	t.Helper()
	m, err := effect.Extract(effect.RestModel{X: 0, Y: 0})
	if err != nil {
		t.Fatalf("effect.Extract: %v", err)
	}
	return m
}

func testCasterModel(t *testing.T, id uint32, luck, hp, maxHp uint16) character.Model {
	t.Helper()
	m, err := character.NewBuilder().
		SetId(id).
		SetLuck(luck).
		SetHp(hp).
		SetMaxHp(maxHp).
		Build()
	if err != nil {
		t.Fatalf("character.NewBuilder().Build(): %v", err)
	}
	return m
}

// seams bundles the overridable dependencies' call counters, restored via
// t.Cleanup so no test leaks state into the next.
type seams struct {
	loadCasterCalls int
	statsCalls      int
	changeHPCalls   int
	changeHPAmount  int16
}

func installSeams(
	t *testing.T,
	caster func(logrus.FieldLogger, context.Context, uint32) (character.Model, error),
	stats func(logrus.FieldLogger, context.Context, field.Model, uint32) (effective_stats.RestModel, error),
	hpErr error,
) *seams {
	t.Helper()
	origCaster, origStats, origHP := loadCaster, loadEffectiveStats, changeHP
	t.Cleanup(func() {
		loadCaster, loadEffectiveStats, changeHP = origCaster, origStats, origHP
	})

	s := &seams{}
	loadCaster = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (character.Model, error) {
		s.loadCasterCalls++
		return caster(l, ctx, characterId)
	}
	loadEffectiveStats = func(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId uint32) (effective_stats.RestModel, error) {
		s.statsCalls++
		return stats(l, ctx, f, characterId)
	}
	changeHP = func(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId uint32, amount int16) error {
		s.changeHPCalls++
		s.changeHPAmount = amount
		return hpErr
	}
	return s
}

// invokeApply opens a recovery window (unless skipWindow), installs seams,
// calls Apply, and returns the seam-call record plus Apply's error. tn and
// charId are caller-chosen so each test gets an isolated registry key on
// the real process-wide singleton.
func invokeApply(
	t *testing.T,
	tn tenant.Model,
	charId uint32,
	skipWindow bool,
	windowY int16,
	caster func(logrus.FieldLogger, context.Context, uint32) (character.Model, error),
	stats func(logrus.FieldLogger, context.Context, field.Model, uint32) (effective_stats.RestModel, error),
	hpErr error,
) (*seams, error) {
	t.Helper()
	ctx := tenant.WithContext(context.Background(), tn)

	if !skipWindow {
		chakrastate.GetRegistry().Start(tn, charId, 30, 99, windowY, time.Now())
	}

	s := installSeams(t, caster, stats, hpErr)
	err := Apply(testLogger())(ctx)(nil, testFieldModel(), charId, testInfo(30), testEffectModel(t))
	return s, err
}

// assertWindowCleared fails the test unless the recovery window for
// tn/charId reads back absent -- i.e. the defer reg.Clear fired.
func assertWindowCleared(t *testing.T, tn tenant.Model, charId uint32) {
	t.Helper()
	if _, ok := chakrastate.GetRegistry().Get(tn, charId, time.Now()); ok {
		t.Fatalf("recovery window for character [%d] still open after Apply; want cleared", charId)
	}
}

// TestApplyNoOpenWindow: the pre-cost gate in character_skill_use.go
// normally rejects this before UseSkill runs; reaching Apply with no open
// window (e.g. the window expired between the two checks) must be a no-op:
// no caster load, no stats lookup, no ChangeHP, no error.
func TestApplyNoOpenWindow(t *testing.T) {
	tn := testTenant(t)
	charId := uint32(9001)

	s, err := invokeApply(t, tn, charId, true, 0,
		func(logrus.FieldLogger, context.Context, uint32) (character.Model, error) {
			t.Fatal("loadCaster should not be called with no open window")
			return character.Model{}, nil
		},
		func(logrus.FieldLogger, context.Context, field.Model, uint32) (effective_stats.RestModel, error) {
			t.Fatal("loadEffectiveStats should not be called with no open window")
			return effective_stats.RestModel{}, nil
		},
		nil,
	)
	if err != nil {
		t.Fatalf("Apply returned unexpected error: %v", err)
	}
	if s.loadCasterCalls != 0 || s.statsCalls != 0 || s.changeHPCalls != 0 {
		t.Fatalf("seam calls = %+v, want all zero for no open window", s)
	}
}

// TestApplyHappyPath: window y=300, effective stats report Luck=100,
// MaxHp=1000 (overriding the caster record's base 0/1000), caster hp=100.
// healDelta(y=300, luck=100, hp=100, maxHp=1000) = 870 (pinned by
// TestHealDelta's "v83 L30 y=300" case) -- ChangeHP must be called with
// exactly that amount, and the window must read back cleared.
func TestApplyHappyPath(t *testing.T) {
	tn := testTenant(t)
	charId := uint32(9002)

	s, err := invokeApply(t, tn, charId, false, 300,
		func(_ logrus.FieldLogger, _ context.Context, id uint32) (character.Model, error) {
			return testCasterModel(t, id, 0, 100, 1000), nil
		},
		func(logrus.FieldLogger, context.Context, field.Model, uint32) (effective_stats.RestModel, error) {
			return effective_stats.RestModel{Luck: 100, MaxHp: 1000}, nil
		},
		nil,
	)
	if err != nil {
		t.Fatalf("Apply returned unexpected error: %v", err)
	}
	if s.loadCasterCalls != 1 || s.statsCalls != 1 {
		t.Fatalf("seam calls = %+v, want loadCaster=1 stats=1", s)
	}
	if s.changeHPCalls != 1 || s.changeHPAmount != 870 {
		t.Fatalf("changeHP calls=%d amount=%d, want calls=1 amount=870", s.changeHPCalls, s.changeHPAmount)
	}
	assertWindowCleared(t, tn, charId)
}

// TestApplyCharacterLoadError: caster load failure logs and returns nil
// (Chakra never surfaces a caster-load failure to the caller -- there is no
// packet to reject at this point, USE_SKILL cost has already been charged),
// emits no ChangeHP, and still clears the window: the window is consumed as
// soon as it is found, before the caster load is attempted.
func TestApplyCharacterLoadError(t *testing.T) {
	tn := testTenant(t)
	charId := uint32(9003)
	loadErr := errors.New("character service unavailable")

	s, err := invokeApply(t, tn, charId, false, 300,
		func(logrus.FieldLogger, context.Context, uint32) (character.Model, error) {
			return character.Model{}, loadErr
		},
		func(logrus.FieldLogger, context.Context, field.Model, uint32) (effective_stats.RestModel, error) {
			t.Fatal("loadEffectiveStats should not be called after a caster load error")
			return effective_stats.RestModel{}, nil
		},
		nil,
	)
	if err != nil {
		t.Fatalf("Apply returned unexpected error: %v", err)
	}
	if s.statsCalls != 0 || s.changeHPCalls != 0 {
		t.Fatalf("seam calls = %+v, want stats=0 changeHP=0 after caster load error", s)
	}
	assertWindowCleared(t, tn, charId)
}

// TestApplyEffectiveStatsFallback: effective-stats lookup fails -> Apply
// falls back to the caster record's base LUK / base MaxHp rather than
// skipping the heal. Caster luck=100, hp=100, maxHp=1000, window y=300 ->
// same 870 as the happy path, computed from base stats instead of live
// ones. ChangeHP still fires and the window still clears.
func TestApplyEffectiveStatsFallback(t *testing.T) {
	tn := testTenant(t)
	charId := uint32(9004)
	statsErr := errors.New("effective-stats unavailable")

	s, err := invokeApply(t, tn, charId, false, 300,
		func(_ logrus.FieldLogger, _ context.Context, id uint32) (character.Model, error) {
			return testCasterModel(t, id, 100, 100, 1000), nil
		},
		func(logrus.FieldLogger, context.Context, field.Model, uint32) (effective_stats.RestModel, error) {
			return effective_stats.RestModel{}, statsErr
		},
		nil,
	)
	if err != nil {
		t.Fatalf("Apply returned unexpected error: %v", err)
	}
	if s.changeHPCalls != 1 || s.changeHPAmount != 870 {
		t.Fatalf("changeHP calls=%d amount=%d, want calls=1 amount=870 (base LUK/MaxHp fallback)", s.changeHPCalls, s.changeHPAmount)
	}
	assertWindowCleared(t, tn, charId)
}

// TestApplyZeroDeltaSkip: caster already at full HP -> healDelta clamps to
// 0 -> ChangeHP must not be emitted, but the window still clears (the defer
// fires unconditionally once the window was found).
func TestApplyZeroDeltaSkip(t *testing.T) {
	tn := testTenant(t)
	charId := uint32(9005)

	s, err := invokeApply(t, tn, charId, false, 300,
		func(_ logrus.FieldLogger, _ context.Context, id uint32) (character.Model, error) {
			return testCasterModel(t, id, 100, 1000, 1000), nil
		},
		func(logrus.FieldLogger, context.Context, field.Model, uint32) (effective_stats.RestModel, error) {
			return effective_stats.RestModel{Luck: 100, MaxHp: 1000}, nil
		},
		nil,
	)
	if err != nil {
		t.Fatalf("Apply returned unexpected error: %v", err)
	}
	if s.changeHPCalls != 0 {
		t.Fatalf("changeHP calls = %d, want 0 at full HP", s.changeHPCalls)
	}
	assertWindowCleared(t, tn, charId)
}

// TestApplyChangeHPError: ChangeHP emit failure is logged and swallowed
// (Apply returns nil), but the window still clears -- it was already
// consumed by the defer before ChangeHP was ever called.
func TestApplyChangeHPError(t *testing.T) {
	tn := testTenant(t)
	charId := uint32(9006)
	hpErr := errors.New("emit failed")

	s, err := invokeApply(t, tn, charId, false, 300,
		func(_ logrus.FieldLogger, _ context.Context, id uint32) (character.Model, error) {
			return testCasterModel(t, id, 0, 100, 1000), nil
		},
		func(logrus.FieldLogger, context.Context, field.Model, uint32) (effective_stats.RestModel, error) {
			return effective_stats.RestModel{Luck: 100, MaxHp: 1000}, nil
		},
		hpErr,
	)
	if err != nil {
		t.Fatalf("Apply returned unexpected error: %v, want nil (ChangeHP errors are logged and swallowed)", err)
	}
	if s.changeHPCalls != 1 || s.changeHPAmount != 870 {
		t.Fatalf("changeHP calls=%d amount=%d, want calls=1 amount=870", s.changeHPCalls, s.changeHPAmount)
	}
	assertWindowCleared(t, tn, charId)
}
