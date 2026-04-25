package job

import (
	constJob "github.com/Chronicle20/atlas/libs/atlas-constants/job"
)

func GetSkillsForJob(jobId uint32) (RestModel, bool) {
	j, ok := constJob.Jobs[constJob.Id(uint16(jobId))]
	if !ok {
		return RestModel{Id: jobId, Skills: []uint32{}}, false
	}
	skills := make([]uint32, 0, len(j.Skills()))
	for _, s := range j.Skills() {
		skills = append(skills, uint32(s.Id()))
	}
	return RestModel{Id: jobId, Skills: skills}, true
}
