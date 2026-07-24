package hide

import (
	"atlas-channel/character"
	"io"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

func tl() logrus.FieldLogger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

func superGm(id uint32) character.Model {
	return character.NewModelBuilder().SetId(id).SetLevel(200).SetJobId(job.SuperGmId).MustBuild()
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
		loadCaster:        func(uint32) (character.Model, error) { return caster, nil },
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
	nonGm := character.NewModelBuilder().SetId(1).SetJobId(job.Id(100)).MustBuild()
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
