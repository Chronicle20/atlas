package character

import "github.com/Chronicle20/atlas/libs/atlas-constants/job"

// WearerProfile carries the non-numeric inputs to equipment requirement checks.
// Lives alongside the numeric stat.Base inside the character Model.
type WearerProfile struct {
	level byte
	jobId job.Id
}

func NewWearerProfile(level byte, jobId job.Id) WearerProfile {
	return WearerProfile{level: level, jobId: jobId}
}

func (p WearerProfile) Level() byte    { return p.level }
func (p WearerProfile) JobId() job.Id  { return p.jobId }
