package mysticdoor

import (
	"context"
	"errors"
	"testing"

	"atlas-channel/data/skill/effect"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/sirupsen/logrus"
)

const (
	testCharId  = uint32(1001)
	testSkillId = uint32(2311002) // PriestMysticDoorId
	testLevel   = byte(1)
	testX       = int16(100)
	testY       = int16(200)
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
		SetSkillId(testSkillId).
		SetSkillLevel(testLevel).
		Build()
}

// invokeApply wires up the three seams and calls Apply; returns whether spawnCalled was set.
func invokeApply(
	t *testing.T,
	mapLoader func(logrus.FieldLogger, context.Context, _map.Id) (uint32, bool, bool, error),
	casterLoader func(logrus.FieldLogger, context.Context, uint32) (int16, int16, error),
	spawnCb func(logrus.FieldLogger, context.Context, field.Model, uint32, uint32, byte, int16, int16) error,
) bool {
	t.Helper()
	origMap := loadMap
	origCaster := loadCaster
	origSpawn := emitSpawn
	t.Cleanup(func() {
		loadMap = origMap
		loadCaster = origCaster
		emitSpawn = origSpawn
	})

	spawnCalled := false
	loadMap = mapLoader
	loadCaster = casterLoader
	emitSpawn = func(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId, skillId uint32, level byte, x, y int16) error {
		spawnCalled = true
		return spawnCb(l, ctx, f, characterId, skillId, level, x, y)
	}

	l := testLogger()
	ctx := context.Background()
	f := testField()
	info := testInfo()
	e := effect.Model{}

	err := Apply(l)(ctx)(nil, f, testCharId, info, e)
	if err != nil {
		t.Fatalf("Apply returned unexpected error: %v", err)
	}
	return spawnCalled
}

func eligibleMapLoader(_ logrus.FieldLogger, _ context.Context, _ _map.Id) (uint32, bool, bool, error) {
	// fieldLimit=0 (no restrictions), town=false, hasReturn=true
	return 0, false, true, nil
}

func eligibleCasterLoader(_ logrus.FieldLogger, _ context.Context, _ uint32) (int16, int16, error) {
	return testX, testY, nil
}

// TestMysticDoorEmitsSpawnWhenEligible: eligible map -> spawn called with caster X/Y.
func TestMysticDoorEmitsSpawnWhenEligible(t *testing.T) {
	var gotX, gotY int16
	called := invokeApply(t,
		eligibleMapLoader,
		eligibleCasterLoader,
		func(_ logrus.FieldLogger, _ context.Context, _ field.Model, characterId, skillId uint32, level byte, x, y int16) error {
			gotX = x
			gotY = y
			return nil
		},
	)
	if !called {
		t.Fatal("emitSpawn was not called for eligible map")
	}
	if gotX != testX || gotY != testY {
		t.Fatalf("emitSpawn called with X=%d Y=%d, want X=%d Y=%d", gotX, gotY, testX, testY)
	}
}

// TestMysticDoorRejectsFieldLimit: fieldLimit&0x02 != 0 -> no spawn.
func TestMysticDoorRejectsFieldLimit(t *testing.T) {
	called := invokeApply(t,
		func(_ logrus.FieldLogger, _ context.Context, _ _map.Id) (uint32, bool, bool, error) {
			return _map.FieldLimitNoMysticDoor, false, true, nil
		},
		eligibleCasterLoader,
		func(_ logrus.FieldLogger, _ context.Context, _ field.Model, _, _ uint32, _ byte, _, _ int16) error {
			return nil
		},
	)
	if called {
		t.Fatal("emitSpawn was called despite FieldLimitNoMysticDoor being set")
	}
}

// TestMysticDoorRejectsTownMap: Town==true -> no spawn.
func TestMysticDoorRejectsTownMap(t *testing.T) {
	called := invokeApply(t,
		func(_ logrus.FieldLogger, _ context.Context, _ _map.Id) (uint32, bool, bool, error) {
			return 0, true, true, nil
		},
		eligibleCasterLoader,
		func(_ logrus.FieldLogger, _ context.Context, _ field.Model, _, _ uint32, _ byte, _, _ int16) error {
			return nil
		},
	)
	if called {
		t.Fatal("emitSpawn was called despite Town==true")
	}
}

// TestMysticDoorRejectsNoReturn: no valid return -> no spawn.
func TestMysticDoorRejectsNoReturn(t *testing.T) {
	called := invokeApply(t,
		func(_ logrus.FieldLogger, _ context.Context, _ _map.Id) (uint32, bool, bool, error) {
			return 0, false, false, nil
		},
		eligibleCasterLoader,
		func(_ logrus.FieldLogger, _ context.Context, _ field.Model, _, _ uint32, _ byte, _, _ int16) error {
			return nil
		},
	)
	if called {
		t.Fatal("emitSpawn was called despite no valid return map")
	}
}

// TestMysticDoorMapLookupError: map load error -> no spawn, no panic.
func TestMysticDoorMapLookupError(t *testing.T) {
	called := invokeApply(t,
		func(_ logrus.FieldLogger, _ context.Context, _ _map.Id) (uint32, bool, bool, error) {
			return 0, false, false, errors.New("map service unavailable")
		},
		eligibleCasterLoader,
		func(_ logrus.FieldLogger, _ context.Context, _ field.Model, _, _ uint32, _ byte, _, _ int16) error {
			return nil
		},
	)
	if called {
		t.Fatal("emitSpawn was called despite map load error")
	}
}
