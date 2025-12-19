package job

import "github.com/Chronicle20/atlas-constants/job"

func JobFromIndex(jobIndex uint32, subJobIndex uint32) job.Id {
	jobId := job.BeginnerId
	if jobIndex == 0 {
		jobId = job.NoblesseId
	} else if jobIndex == 1 {
		if subJobIndex == 0 {
			jobId = job.BeginnerId
		} else if subJobIndex == 1 {
			//jobId = job.BladeRecruit TODO
		}
	} else if jobIndex == 2 {
		jobId = job.LegendId
	} else if jobIndex == 3 {
		jobId = job.EvanId
	}
	return jobId
}
