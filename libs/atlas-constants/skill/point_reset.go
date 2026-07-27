package skill

// IsPointResetExcluded reports whether skillId may not participate in an SP
// Reset transfer (as source or target): Aran hidden combo skills, GM skills,
// and PQ-granted skills, whose points are not pool-backed
// (see docs/tasks/task-126-ap-sp-reset-items/design.md §4.1).
func IsPointResetExcluded(skillId Id) bool {
	switch skillId {
	case Id(21110007), Id(21110008), Id(21120009), Id(21120010): // Aran hidden combo
		return true
	case Id(10000013), Id(20001013): // PQ skills (fixed ids)
		return true
	}
	if skillId >= Id(9001000) && skillId <= Id(9101008) { // GM skills
		return true
	}
	if skillId >= Id(8001000) && skillId <= Id(8001001) { // GM skills
		return true
	}
	if skillId >= Id(20000014) && skillId <= Id(20000018) { // PQ skills
		return true
	}
	rem := uint32(skillId) % 10000000
	if rem >= 1009 && rem <= 1011 { // PQ skills (per-class beginner band)
		return true
	}
	if rem == 1020 { // PQ skill
		return true
	}
	return false
}
