package model

// NewSkillUsageInfoForTest constructs a SkillUsageInfo with the supplied
// fields. Only consumers in test code should use this — production code
// must populate SkillUsageInfo through the wire decoder.
func NewSkillUsageInfoForTest(skillId uint32, level byte, affectedMobIds []uint32) SkillUsageInfo {
	return SkillUsageInfo{
		skillId:        skillId,
		skillLevel:     level,
		affectedMobIds: affectedMobIds,
	}
}
