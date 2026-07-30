// Package constants is the atlas-constants tenant-keyed selector
// (task-187 Task 6): the single entry point services use to get a tenant
// version's skill/job identity Sets, layered on top of the per-version
// registry generated from Tasks 2-5.
package constants

import (
	"strings"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

// SkillJobSet bundles one tenant version's skill and job identity Sets --
// the pair callers actually need to resolve/wire both domains for a given
// (region, major, minor).
type SkillJobSet struct {
	Skill skill.Set
	Job   job.Set
}

// loggedMisses de-duplicates the unprovisioned-version warning below: an
// unprovisioned version falling back on every call (e.g. once per packet)
// must not spam the log -- only the first miss per (region,major,minor)
// tuple is logged.
var loggedMisses sync.Map // versionKey -> struct{}

// For returns the SkillJobSet for a tenant's (region, major, minor). region
// is matched case-insensitively (tenant.Region() returns upper-case
// "GMS"/"JMS", but callers should not have to worry about case).
//
// If the tuple is not a provisioned registry entry (see
// constants/registry_gen.go and docs/tasks/task-187-version-aware-id-semantics),
// For falls back to the canonical GMS 83.1 baseline and logs a warning --
// once per unprovisioned tuple, not once per call.
func For(region string, major, minor uint16) SkillJobSet {
	k := versionKey{region: strings.ToUpper(region), major: major, minor: minor}
	if s, ok := registry[k]; ok {
		return s
	}

	if _, dup := loggedMisses.LoadOrStore(k, struct{}{}); !dup {
		logrus.StandardLogger().WithFields(logrus.Fields{
			"region": k.region,
			"major":  major,
			"minor":  minor,
		}).Warn("constants.For: unprovisioned version; using GMS 83.1 baseline")
	}
	return baseline
}
