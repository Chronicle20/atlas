package skill

import (
	"sync"

	"github.com/sirupsen/logrus"
)

// Stats counts one ingest pass's skill derivation outcomes (FR-7.3). The
// counters are an explicit return value rather than a package-level registry:
// they are per-run, not per-tenant-lifetime, and hidden global mutable state
// makes unit tests order-dependent.
type Stats struct {
	Processed          int
	FromCommon         int
	FromLevel          int
	Neither            int
	SkillsWithFailures int
	Failures           int
}

func (s *Stats) Add(o Stats) {
	s.Processed += o.Processed
	s.FromCommon += o.FromCommon
	s.FromLevel += o.FromLevel
	s.Neither += o.Neither
	s.SkillsWithFailures += o.SkillsWithFailures
	s.Failures += o.Failures
}

// Derivation is what reading one Skill.wz job image produces: the skill
// documents and the counters describing how they were derived.
type Derivation struct {
	Models []RestModel
	Stats  Stats
}

// StatsAccumulator sums Stats across the job images of one ingest run and
// emits the run summary. Both ingest entry points (the SKILL worker and the
// legacy data processor) use it, so neither can silently drop the summary.
type StatsAccumulator struct {
	mu    sync.Mutex
	total Stats
}

// Wrap adapts a stats-returning register function to the plain
// func(path) error shape the directory walkers expect, accumulating as it
// goes. A failing register contributes nothing to the totals.
func (a *StatsAccumulator) Wrap(rf func(path string) (Stats, error)) func(path string) error {
	return func(path string) error {
		s, err := rf(path)
		if err != nil {
			return err
		}
		a.mu.Lock()
		a.total.Add(s)
		a.mu.Unlock()
		return nil
	}
}

// Total returns a copy of the accumulated counters.
func (a *StatsAccumulator) Total() Stats {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.total
}

// Log emits the FR-7.3 run summary. The separate ERROR line is what makes the
// per-failure permissiveness safe: one ERROR can scroll past in a 954-skill
// run, an aggregate at the end cannot.
func (a *StatsAccumulator) Log(l logrus.FieldLogger) {
	s := a.Total()
	l.Infof("skills: processed=%d fromCommon=%d fromLevel=%d neither=%d failures=%d",
		s.Processed, s.FromCommon, s.FromLevel, s.Neither, s.Failures)
	if s.SkillsWithFailures > 0 {
		l.WithFields(logrus.Fields{
			"skillsWithFailures": s.SkillsWithFailures,
			"failures":           s.Failures,
		}).Errorf("Skill.wz ingest had skills with common-node derivation failures.")
	}
}
