package skill

// IsTamedMountSkill reports whether id is a tamed-monster MonsterRider skill
// (Beginner/Noblesse/Legend/Evan band). Tamed mounts read the equipped taming-mob
// item id as the vehicle; skill-only mounts do not.
func IsTamedMountSkill(id Id) bool {
	return Is(id, BeginnerMonsterRidingId, NoblesseMonsterRidingId, LegendMonsterRidingId, EvanMonsterRidingId)
}

// SkillOnlyMountVehicleId maps a skill-only mount skill id (any band) to its
// fixed vehicle item id. SpaceShip is per-level (1932000+level). Returns false
// for ids that are not skill-only mounts.
func SkillOnlyMountVehicleId(id Id, level int) (int32, bool) {
	switch id {
	case BeginnerSpaceShipId, NoblesseSpaceShipId:
		return int32(1932000 + level), true
	case BeginnerYetiMount1Id, NoblesseYetiMount1Id, LegendYetiMount1Id:
		return 1932003, true
	case BeginnerYetiMount2Id, NoblesseYetiMount2Id, LegendYetiMount2Id:
		return 1932004, true
	case BeginnerBroomstickId, NoblesseBroomstickId, LegendBroomstickId:
		return 1932005, true
	case BeginnerBalrogMountId, NoblesseBalrogMountId, LegendBalrogMountId:
		return 1932010, true
	default:
		return 0, false
	}
}
