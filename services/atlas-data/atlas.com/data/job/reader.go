package job

import (
	"atlas-data/xml"
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// parseJobId derives the job id from a Skill.wz image name. The name reaches
// this helper as the root <imgdir name="…"> attribute (e.g. "112.img"), which
// is the mapping FR-2.2 requires: the job id is read off the image, never
// derived by dividing skill ids. Duplicated rather than exported from `skill`
// so that `job` and `skill` stay dependency-free in this direction (D1 — the
// list resource makes `job` import `skill`, so `skill` must not import `job`).
func parseJobId(name string) (uint32, error) {
	baseName := filepath.Base(name)
	if !strings.HasSuffix(baseName, ".img") {
		return 0, fmt.Errorf("file does not match expected format: %s", name)
	}
	idStr := strings.TrimSuffix(baseName, ".img")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, err
	}
	return uint32(id), nil
}

// Read produces the JOB document for one Skill.wz image: exactly one model for
// a per-job (numeric) image, none for a non-numeric one such as MobSkill.img
// or BFSkill.img.
//
// Two deliberate divergences from skill.Read (design D5):
//   - A non-numeric image yields an empty slice and NO error, so the new
//     registration pass adds no `register MobSkill.img.xml` warn noise to the
//     SKILL worker (see data/workers/walk.go:45-47).
//   - A missing or empty `skill` child yields a model with an empty skill
//     list. FR-2.4 requires "the job exists with zero skills" to be
//     representable and distinguishable from "the job is absent".
//
// Skill ids are emitted in WZ document order, not sorted (FR-1.2): the order is
// deterministic per archive, so re-ingest is byte-stable and baseline dumps do
// not churn.
//
// ctx is accepted for signature parity with skill.Read and the shared
// Register plumbing; the JOB reader itself needs no tenant.
func Read(l logrus.FieldLogger) func(ctx context.Context) func(np model.Provider[xml.Node]) model.Provider[[]RestModel] {
	return func(ctx context.Context) func(np model.Provider[xml.Node]) model.Provider[[]RestModel] {
		return func(np model.Provider[xml.Node]) model.Provider[[]RestModel] {
			exml, err := np()
			if err != nil {
				return model.ErrorProvider[[]RestModel](err)
			}

			jobId, err := parseJobId(exml.Name)
			if err != nil {
				// Not a per-job image (MobSkill.img, BFSkill.img, ...). FR-2.3.
				return model.FixedProvider([]RestModel{})
			}

			skills := make([]uint32, 0)
			if ssxml, err := exml.ChildByName("skill"); err == nil {
				for _, sxml := range ssxml.ChildNodes {
					skillId, err := strconv.ParseUint(sxml.Name, 10, 32)
					if err != nil {
						continue
					}
					skills = append(skills, uint32(skillId))
				}
			}
			l.Debugf("Read [%d] skills for job [%d].", len(skills), jobId)

			return model.FixedProvider([]RestModel{{Id: jobId, Skills: skills}})
		}
	}
}
