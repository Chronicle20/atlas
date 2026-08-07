package mprecovery

import (
	"atlas-channel/data/skill/effect"
	"context"
	"errors"
	"testing"

	channelhandler "atlas-channel/skill/handler"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

const (
	testCharId = uint32(1001)
	testLevel  = byte(5)
	testMaxHp  = uint16(1234)
)

func testLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetLevel(logrus.DebugLevel)
	return l
}

func testField() field.Model {
	return field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).Build()
}

func testInfo() packetmodel.SkillUsageInfo {
	return packetmodel.NewSkillUsageInfoBuilder().
		SetSkillId(uint32(skill2.BrawlerMPRecoveryId)).
		SetSkillLevel(testLevel).
		Build()
}

// testEffect builds an effect.Model with the given x/y through the REST
// extract path (no builder exists on the model; Extract is the production
// construction seam and exercises the Y() getter end-to-end).
func testEffect(t *testing.T, x int16, y int16) effect.Model {
	t.Helper()
	e, err := effect.Extract(effect.RestModel{X: x, Y: y})
	if err != nil {
		t.Fatalf("effect.Extract returned error: %v", err)
	}
	return e
}

// call records one seam invocation, preserving order across seams.
type call struct {
	name   string
	amount int16
}

// invokeApply overrides the three seams, calls Apply, and returns the
// ordered seam calls plus Apply's error.
func invokeApply(
	t *testing.T,
	casterLoader func(logrus.FieldLogger, context.Context, uint32) (uint16, error),
	hpErr error,
	mpErr error,
	e effect.Model,
) ([]call, error) {
	t.Helper()
	origCaster, origHP, origMP := loadCaster, changeHP, changeMP
	t.Cleanup(func() {
		loadCaster, changeHP, changeMP = origCaster, origHP, origMP
	})

	var calls []call
	loadCaster = casterLoader
	changeHP = func(_ logrus.FieldLogger, _ context.Context, _ field.Model, _ uint32, amount int16) error {
		calls = append(calls, call{name: "changeHP", amount: amount})
		return hpErr
	}
	changeMP = func(_ logrus.FieldLogger, _ context.Context, _ field.Model, _ uint32, amount int16) error {
		calls = append(calls, call{name: "changeMP", amount: amount})
		return mpErr
	}

	err := Apply(testLogger())(context.Background())(nil, testField(), testCharId, testInfo(), e)
	return calls, err
}

func happyCasterLoader(_ logrus.FieldLogger, _ context.Context, _ uint32) (uint16, error) {
	return testMaxHp, nil
}

// TestMPRecoveryRegistered: init() installs Apply in the shared registry.
func TestMPRecoveryRegistered(t *testing.T) {
	h, ok := channelhandler.Lookup(skill2.BrawlerMPRecovery)
	if !ok || h == nil {
		t.Fatalf("Lookup(BrawlerMPRecovery) = (%v, %v), want non-nil handler", h, ok)
	}
}

// TestMPRecoveryHappyPath: v83 L5 (x=10, y=75), MaxHp 1234 -> ChangeHP(-123)
// then ChangeMP(+92), in that order. The handler has no currentHP input, so
// this also pins FR-3: the full unclamped loss is emitted regardless of the
// caster's current HP (atlas-character owns the 0-floor/death path).
func TestMPRecoveryHappyPath(t *testing.T) {
	calls, err := invokeApply(t, happyCasterLoader, nil, nil, testEffect(t, 10, 75))
	if err != nil {
		t.Fatalf("Apply returned unexpected error: %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("got %d seam calls %v, want 2 (changeHP then changeMP)", len(calls), calls)
	}
	if calls[0].name != "changeHP" || calls[0].amount != -123 {
		t.Fatalf("first call = %+v, want changeHP with -123", calls[0])
	}
	if calls[1].name != "changeMP" || calls[1].amount != 92 {
		t.Fatalf("second call = %+v, want changeMP with +92", calls[1])
	}
}

// TestMPRecoveryCasterLoadError: FR-5 — load failure emits nothing and
// surfaces the error.
func TestMPRecoveryCasterLoadError(t *testing.T) {
	loadErr := errors.New("character service unavailable")
	calls, err := invokeApply(t,
		func(_ logrus.FieldLogger, _ context.Context, _ uint32) (uint16, error) {
			return 0, loadErr
		},
		nil, nil, testEffect(t, 10, 75))
	if !errors.Is(err, loadErr) {
		t.Fatalf("Apply error = %v, want %v", err, loadErr)
	}
	if len(calls) != 0 {
		t.Fatalf("got seam calls %v, want none on caster load failure", calls)
	}
}

// TestMPRecoveryChangeHPError: FR-5 — never MP gain without the HP cost
// having been requested; a ChangeHP emit failure skips ChangeMP.
func TestMPRecoveryChangeHPError(t *testing.T) {
	hpErr := errors.New("emit failed")
	calls, err := invokeApply(t, happyCasterLoader, hpErr, nil, testEffect(t, 10, 75))
	if !errors.Is(err, hpErr) {
		t.Fatalf("Apply error = %v, want %v", err, hpErr)
	}
	if len(calls) != 1 || calls[0].name != "changeHP" {
		t.Fatalf("got seam calls %v, want exactly one changeHP call", calls)
	}
}

// TestMPRecoveryChangeMPError: the HP cost was already requested (the HP
// cost is applied before MP gain); the MP error is surfaced.
func TestMPRecoveryChangeMPError(t *testing.T) {
	mpErr := errors.New("emit failed")
	calls, err := invokeApply(t, happyCasterLoader, nil, mpErr, testEffect(t, 10, 75))
	if !errors.Is(err, mpErr) {
		t.Fatalf("Apply error = %v, want %v", err, mpErr)
	}
	if len(calls) != 2 {
		t.Fatalf("got seam calls %v, want changeHP then changeMP", calls)
	}
}

// TestMPRecoveryBadDataSkips: x=0 (bad tenant data) -> warn, nil error,
// zero emits. Never divide by zero, never emit a zero delta.
func TestMPRecoveryBadDataSkips(t *testing.T) {
	calls, err := invokeApply(t, happyCasterLoader, nil, nil, testEffect(t, 0, 75))
	if err != nil {
		t.Fatalf("Apply returned unexpected error: %v", err)
	}
	if len(calls) != 0 {
		t.Fatalf("got seam calls %v, want none for x=0", calls)
	}
}

// TestMPRecoveryZeroMpGainSkipsChangeMP: hpLost > 0 but mpGain floors to 0
// (MaxHp 10, x=10, y=75 -> hpLost 1, mpGain 0) — the HP cost still applies,
// the zero-delta MP emit is skipped.
func TestMPRecoveryZeroMpGainSkipsChangeMP(t *testing.T) {
	calls, err := invokeApply(t,
		func(_ logrus.FieldLogger, _ context.Context, _ uint32) (uint16, error) {
			return 10, nil
		},
		nil, nil, testEffect(t, 10, 75))
	if err != nil {
		t.Fatalf("Apply returned unexpected error: %v", err)
	}
	if len(calls) != 1 || calls[0].name != "changeHP" || calls[0].amount != -1 {
		t.Fatalf("got seam calls %v, want exactly changeHP with -1", calls)
	}
}
