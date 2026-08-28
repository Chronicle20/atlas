package job

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
