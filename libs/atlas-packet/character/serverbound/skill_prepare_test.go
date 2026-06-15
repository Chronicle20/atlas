package serverbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// buildSampleSkillPrepare mirrors model.sampleSkillPrepareInfo: a standard keydown
// skill (Hurricane 3121004) so the swallowMobId branch (skillId == 33101005) stays
// quiet and the wire structure is driven purely by tenant version.
func buildSampleSkillPrepare() model.SkillPrepareInfo {
	m := model.NewSkillPrepareInfo()
	m.SetSkillId(3121004)
	m.SetLevel(10)
	m.SetAction(0x0142)
	m.SetActionSpeed(4)
	return *m
}

// TestSkillPrepareRoundTrip pins that the serverbound SkillPrepare wrapper delegates
// symmetrically to the shared model.SkillPrepareInfo codec across all tenant variants.
// The model itself (incl. the swallowMobId branch) is production-tested in
// model/skill_prepare_info_test.go.
func TestSkillPrepareRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			m := SkillPrepare{info: buildSampleSkillPrepare()}
			pt.RoundTrip(t, ctx, m.Encode, m.Decode, nil)
		})
	}
}
