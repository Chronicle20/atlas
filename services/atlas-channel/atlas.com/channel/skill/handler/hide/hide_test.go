package hide

import (
	"atlas-channel/character"
	"io"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/constants"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

func tl() logrus.FieldLogger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

func superGm(id uint32) character.Model {
	return character.NewBuilder().SetId(id).SetLevel(200).SetJobId(job.SuperGmId).MustBuild()
}

type hideCapture struct {
	applied   int
	cancelled int
	despawned int
	spawned   int
	self      int
}

func deps(caster character.Model, hidden bool, c *hideCapture) hideDeps {
	return hideDeps{
		loadCaster: func(uint32) (character.Model, error) { return caster, nil },
		// isSuperGm mirrors the pre-task-187 version-blind comparison: these
		// unit tests exercise applyHide's core loop with canonical (v83-era)
		// job ids directly, not tenant-version resolution -- that property is
		// pinned separately (skill/handler/version_resolve_test.go's v48/v83
		// IsSuperGm tests, and TestResolveHideSourceId_v48VsV83 below).
		isSuperGm:         func(jid job.Id) bool { return job.IsA(jid, job.SuperGmId) },
		isHidden:          func(uint32) (bool, error) { return hidden, nil },
		applyHide:         func(field.Model, uint32, byte) error { c.applied++; return nil },
		cancelHide:        func(field.Model, uint32) error { c.cancelled++; return nil },
		despawnFromOthers: func(field.Model, uint32) error { c.despawned++; return nil },
		spawnToOthers:     func(field.Model, uint32) error { c.spawned++; return nil },
		announceSelf:      func(byte) error { c.self++; return nil },
	}
}

func info() packetmodel.SkillUsageInfo { return packetmodel.SkillUsageInfo{} } // SkillLevel() -> 0 is fine

func TestNonSuperGmRejected(t *testing.T) {
	nonGm := character.NewBuilder().SetId(1).SetJobId(job.Id(100)).MustBuild()
	var c hideCapture
	_ = applyHide(tl(), field.NewBuilder(0, 0, 1).Build(), 1, info(), deps(nonGm, false, &c))
	if c.applied+c.cancelled+c.despawned+c.spawned != 0 {
		t.Errorf("non-SuperGM caster produced effects: %+v", c)
	}
}

func TestHideOn(t *testing.T) {
	var c hideCapture
	_ = applyHide(tl(), field.NewBuilder(0, 0, 1).Build(), 1, info(), deps(superGm(1), false, &c))
	if c.applied != 1 || c.despawned != 1 {
		t.Errorf("hide ON: applied=%d despawned=%d, want 1/1", c.applied, c.despawned)
	}
	if c.cancelled != 0 || c.spawned != 0 {
		t.Errorf("hide ON leaked cancel/spawn: %+v", c)
	}
	if c.self != 1 {
		t.Errorf("hide ON self-announce=%d, want 1", c.self)
	}
}

func TestHideOff(t *testing.T) {
	var c hideCapture
	_ = applyHide(tl(), field.NewBuilder(0, 0, 1).Build(), 1, info(), deps(superGm(1), true, &c))
	if c.cancelled != 1 || c.spawned != 1 {
		t.Errorf("hide OFF: cancelled=%d spawned=%d, want 1/1", c.cancelled, c.spawned)
	}
	if c.applied != 0 || c.despawned != 0 {
		t.Errorf("hide OFF leaked apply/despawn: %+v", c)
	}
}

// TestResolveHideSourceId_v48VsV83 pins the outbound half of the task-187
// v0.48 correctness fix: the GM-hide buff's SourceId must be the CASTER'S
// VERSION's wire id for SuperGmHide, not the hardcoded canonical (v83-era)
// wire value -- otherwise a v0.48 hide buff would carry SourceId 9101004 (not
// a valid v0.48 wire id at all), and character/buff.IsGmHidden's
// version-aware resolve (which resolves the SAME tenant's version set) would
// never recognize it as a hide buff.
func TestResolveHideSourceId_v48VsV83(t *testing.T) {
	if got := resolveHideSourceId(constants.For("GMS", 48, 1)); got != 5101004 {
		t.Errorf("resolveHideSourceId(v48) = %d, want 5101004", got)
	}
	if got := resolveHideSourceId(constants.For("GMS", 83, 1)); got != int32(skill2.SuperGmHideId) {
		t.Errorf("resolveHideSourceId(v83) = %d, want %d (canonical)", got, skill2.SuperGmHideId)
	}
}
