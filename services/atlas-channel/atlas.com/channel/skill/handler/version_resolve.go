package handler

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/constants"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
)

// IsSuperGm reports whether jobId -- a version-specific WIRE job id, as
// character.Model.JobId() always is -- resolves to the SuperGM job identity
// under set. Shared by the GM-only per-skill handlers (hide, healdispel):
// job.SuperGmId is the canonical (v83-era) wire token 910, but at v0.48
// SuperGM is wire 510, so a raw `job.IsA(c.JobId(), job.SuperGmId)` compare
// silently rejects every v0.48 GM cast (task-187).
func IsSuperGm(set constants.SkillJobSet, jobId job.Id) bool {
	jid, ok := set.Job.Resolve(jobId)
	return ok && job.IsAIdentity(jid, job.SuperGm)
}
