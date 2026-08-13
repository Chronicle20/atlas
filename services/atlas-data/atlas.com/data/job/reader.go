package job

import (
	"atlas-data/skill"
	"atlas-data/xml"
	"context"
	"strconv"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// Read produces the JOB document for one Skill.wz image: exactly one model for
// a per-job (numeric) image, none for a non-numeric one such as MobSkill.img
// or BFSkill.img.
//
// Two deliberate divergences from skill.Read (design D5):
//   - A non-numeric image yields an empty slice and NO error, so the new
//     registration pass adds no `register MobSkill.img.xml` warn noise to the
//     SKILL worker (see data/workers/walk.go:45-47).
//   - An ABSENT `skill` child yields NO model (task-202 FR-1.1): the image is
//     not a job document. A PRESENT but empty `skill` child yields a model with
//     an empty skill list (FR-1.2) -- "the job exists with zero skills" stays
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

			jobId, err := skill.ParseJobId(exml.Name)
			if err != nil {
				// Not a per-job image (MobSkill.img, BFSkill.img, ...). FR-2.3.
				return model.FixedProvider([]RestModel{})
			}

			ssxml, err := exml.ChildByName("skill")
			if err != nil {
				// FR-1.1 (task-202): no `skill` node at all means this image is
				// not a job document. Skill.wz/Dragon/ at v0.84+ holds Evan's
				// Mir ANIMATION images named exactly 2200.img-2218.img; emitting
				// an empty model for them let document upsert (last-write-wins
				// on (tenant, type, document_id)) blank the real Evan documents.
				// Returning no model makes the outcome independent of walk order.
				//
				// A PRESENT but empty `skill` node is the opposite case and
				// still yields a model (FR-1.2): 1112.img (Cygnus 4th job) is a
				// real job with zero skills. Never collapse these two into a
				// len(skills) == 0 check.
				l.Debugf("Image [%s] has no skill node; not a JOB document.", exml.Name)
				return model.FixedProvider([]RestModel{})
			}

			skills := make([]uint32, 0)
			for _, sxml := range ssxml.ChildNodes {
				skillId, err := strconv.ParseUint(sxml.Name, 10, 32)
				if err != nil {
					continue
				}
				skills = append(skills, uint32(skillId))
			}
			l.Debugf("Read [%d] skills for job [%d].", len(skills), jobId)

			return model.FixedProvider([]RestModel{{Id: jobId, Skills: skills}})
		}
	}
}
