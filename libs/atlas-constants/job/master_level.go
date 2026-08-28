package job

import (
	"strings"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

// ClientJobLevel is a verbatim port of the client's get_job_level(nJob)
// (GMS v92 sub_479260 @0x479260, GMS v95 @0x47cb90, JMS 185 @0x47d347).
//
// It is NOT Advancement. Advancement returns -1 for the whole Evan stage
// line and has no 43x branch; the client's helper returns 2..10 for Evan
// growths and 2..4 for Dual Blades, and both of those are load-bearing for
// is_skill_need_master_level. Reusing Advancement here would be the same
// class of mistake as the job.GetSkillBook off-by-one that task-218 hit.
//
//	if job%100 == 0 || job == 2001: return 1
//	v := job%10;  if job/10 == 43 { v = (job-430)/2 }
//	lvl := v + 2
//	return lvl if lvl >= 2 && (lvl <= 4 || (lvl <= 10 && isEvanJob(job))) else 0
//
// The 43x expression differs between regions — GMS computes (job-430)/2,
// JMS 185 (@0x47d347) computes (job%10)/2 — but the two agree over the whole
// defined range 430..434 (0,0,1,1,2) and diverge only at job >= 435, which no
// client defines. Modelled as the GMS form; do not branch on region here.
func ClientJobLevel(jobId Id) int {
	if jobId%100 == 0 || jobId == 2001 {
		return 1
	}
	v := int(jobId % 10)
	if jobId/10 == 43 {
		v = (int(jobId) - 430) / 2
	}
	lvl := v + 2
	if lvl < 2 {
		return 0
	}
	if lvl <= 4 {
		return lvl
	}
	if lvl <= 10 && isEvanJob(jobId) {
		return lvl
	}
	return 0
}

// isEvanJob is the client's is_evan_job (GMS v95 @0x47cad0): the Evan growth
// band plus the Evan beginner. JMS 185 inlines only job/100 == 22, which is
// equivalent at its call sites because job == 2001 has already returned.
func isEvanJob(jobId Id) bool {
	return jobId/100 == 22 || jobId == 2001
}

// dualBladeShape is which of the three forms a client's 430-434 arm takes.
type dualBladeShape int

const (
	// dualBladeNone: no 43x arm at all; the id falls through to the common
	// rule, which is TRUE for job 432. GMS v83 @0x4e8f04, v84 @0x4f0ad2.
	dualBladeNone dualBladeShape = iota
	// dualBladeAlwaysFalse: the arm exists and returns 0 for all of 430-434.
	// GMS v87 @0x508fa4 (v1/10 == 43 || !(v1%100) -> return 0).
	dualBladeAlwaysFalse
	// dualBladeJobLevel: ClientJobLevel(job) == 4, or one of four named ids.
	// GMS v92 @0x479371, GMS v95 @0x47ccb0, JMS 185 @0x47d2f9.
	dualBladeJobLevel
)

// dualBladeArm reports which shape this client's 430-434 arm takes. The arms
// are monotone in major within a region, so an unprovisioned major takes the
// nearest lower provisioned arm rather than a baseline fallback.
func dualBladeArm(region string, major uint16) dualBladeShape {
	if strings.ToUpper(region) != "GMS" {
		return dualBladeJobLevel
	}
	switch {
	case major < 87:
		return dualBladeNone
	case major < 92:
		return dualBladeAlwaysFalse
	default:
		return dualBladeJobLevel
	}
}

// hasEvanExceptions reports whether this client's Evan arm carries the
// three-skill exception list {22111001, 22141002, 22140000}. Present on
// GMS v84 (@0x4f0ad2), v87 (@0x508f33), v92 (@0x4792f0) and v95 (@0x47ccb0);
// absent on GMS v83 (@0x4e8f04) and on JMS 185 (@0x47d2a8).
func hasEvanExceptions(region string, major uint16) bool {
	return strings.ToUpper(region) == "GMS" && major >= 84
}

// ignoresCommonMasterLevel reports whether this client early-outs on
// is_ignore_master_level_for_common. GMS v95 @0x47cc20 is the only client
// Atlas has read that carries it; >= 95 rather than == 95 because the arms
// are monotone in major within a region.
func ignoresCommonMasterLevel(region string, major uint16) bool {
	return strings.ToUpper(region) == "GMS" && major >= 95
}

// ignoredCommonMasterLevelSkills is the flat membership of GMS v95's
// is_ignore_master_level_for_common (@0x47cc20). Its callers see a false
// is_skill_need_master_level for every member.
//
// Fourteen of the sixteen belong to jobs an Atlas tenant can create
// (112, 122, 132, 212, 222, 232, 312, 322, 412, 422, 512, 522). The other
// two (jobs 3212, 3312) match no Atlas identity and are modelled only
// because the client's list is a flat id set and a partial port is not one.
var ignoredCommonMasterLevelSkills = map[skill.Id]struct{}{
	1120012: {}, 1220013: {}, 1320011: {},
	2120009: {}, 2220009: {}, 2320010: {},
	3120010: {}, 3120011: {}, 3220009: {}, 3220010: {},
	4120010: {}, 4220009: {},
	5120011: {}, 5220012: {},
	32120009: {}, 33120010: {},
}

// NeedsMasterLevel reports whether a skill entry in GW_CharacterData carries
// the trailing 4-byte master level. It is a direct port of the client's
// is_skill_need_master_level(nSkillID) (GMS v83 @0x4e8f04, v84 @0x4f0ad2,
// v87 @0x508f33, v92 @0x4792f0, v95 @0x47ccb0, JMS 185 @0x47d2a8), which is
// the ONLY authority: the field is per-SKILL, and approximating it with a
// per-JOB test is what produced the task-218 field report (a preset Evan
// closed the client with error 38 while a level-1 Evan logged in fine,
// because the length only diverges once the character owns skills).
//
// Because the field is not length-prefixed, answering it differently than
// the client does not produce a wrong value — it shifts every subsequent
// field of GW_CharacterData by four bytes.
//
// region/major select the arms; region is matched case-insensitively, the
// same normalisation constants.For applies (constants/for.go:40). The arms
// are monotone in major within a region, so an unprovisioned major takes the
// nearest lower provisioned arm; there is no baseline fallback and no
// logging, because this runs once per skill inside an encode loop.
func NeedsMasterLevel(skillId skill.Id, region string, major uint16) bool {
	if ignoresCommonMasterLevel(region, major) {
		if _, ok := ignoredCommonMasterLevelSkills[skillId]; ok {
			return false
		}
	}

	jobId := Id(uint32(skillId) / 10000)

	if isEvanJob(jobId) {
		// Growths 9 and 10 (jobs 2217, 2218) on every client read.
		if lvl := ClientJobLevel(jobId); lvl == 9 || lvl == 10 {
			return true
		}
		if !hasEvanExceptions(region, major) {
			return false
		}
		return skill.Is(skillId, skill.Id(22111001), skill.Id(22141002), skill.Id(22140000))
	}

	if jobId/10 == 43 {
		switch dualBladeArm(region, major) {
		case dualBladeAlwaysFalse:
			return false
		case dualBladeJobLevel:
			return ClientJobLevel(jobId) == 4 ||
				skill.Is(skillId, skill.Id(4311003), skill.Id(4321000), skill.Id(4331002), skill.Id(4331005))
		case dualBladeNone:
			// No arm: fall through to the common rule below, which is what
			// GMS v83/v84 do — and which is TRUE for job 432.
		}
	}

	if jobId%100 == 0 {
		return false
	}
	return jobId%10 == 2
}
