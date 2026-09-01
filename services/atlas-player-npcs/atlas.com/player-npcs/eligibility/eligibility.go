// Package eligibility is the single predicate FR-1.1's automatic deploy
// check and FR-6.1's conversation condition both evaluate (design §9.1) so
// the two can never disagree.
package eligibility

import (
	"atlas-player-npcs/character"
	"atlas-player-npcs/configuration"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
)

// Failure codes (design §8.3).
const (
	ReasonIneligible = "ineligible"
	ReasonDuplicate  = "duplicate"
)

// Evaluate reports whether char may have a Player NPC deployed, and if
// not, the design §8.3 reason code.
//
// The base predicate (both callers): level >= the job's max level
// (FR-1.2, job.MaxLevelFor), not a GM, and no existing Player NPC already
// deployed for (character name, map) -- existingCount is that count,
// computed by the caller.
//
// conversationPath is true only for the conversation engine's
// canSpawnPlayerNpc condition (FR-6.1); it adds the requirement that
// automatic deployment is disabled for the tenant, per cfg. The automatic
// deploy check (FR-1.1) passes conversationPath false and so does not
// constrain cfg.AutoDeployEnabled() at all -- it is only reached when
// automatic deployment already fired.
func Evaluate(cfg configuration.Model, char character.Model, existingCount int, conversationPath bool) (bool, string) {
	if conversationPath && cfg.AutoDeployEnabled() {
		return false, ReasonIneligible
	}
	if char.Level() < job.MaxLevelFor(char.JobId()) {
		return false, ReasonIneligible
	}
	if char.Gm() {
		return false, ReasonIneligible
	}
	if existingCount > 0 {
		return false, ReasonDuplicate
	}
	return true, ""
}
