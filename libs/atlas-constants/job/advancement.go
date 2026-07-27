package job

// Advancement returns the job-advancement tier (0-4) for a job id:
// 0 for beginners (Beginner/Noblesse/Legend/Evan-beginner), 1 for a branch
// root (jobId%100 == 0), else 2 + jobId%10. Evan stage lines (2200-2218) do
// not map onto the 4-tier scheme and return -1, as does any id whose derived
// tier falls outside 0-4.
func Advancement(jobId Id) int {
	if jobId >= EvanStage1Id && jobId <= EvanStage10Id {
		return -1
	}
	if IsBeginner(jobId) {
		return 0
	}
	if jobId%100 == 0 {
		return 1
	}
	tier := 2 + int(jobId%10)
	if tier > 4 {
		return -1
	}
	return tier
}
