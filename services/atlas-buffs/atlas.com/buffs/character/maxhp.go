package character

import (
	"atlas-buffs/berserk"
	"atlas-buffs/buff/stat"
	"context"

	"github.com/sirupsen/logrus"

	constants "github.com/Chronicle20/atlas/libs/atlas-constants/character"
)

// maxHpBuffStatTypes mirrors the MapBuffStatType cases in atlas-effective-stats
// that resolve to its max-HP stat (services/atlas-effective-stats/atlas.com/
// effective-stats/stat/model.go — currently only HYPER_BODY_HP). Keep in sync
// if effective-stats grows new max-HP-affecting buff mappings.
var maxHpBuffStatTypes = map[string]bool{
	string(constants.TemporaryStatTypeHyperBodyHP): true,
}

func affectsMaxHp(changes []stat.Model) bool {
	for _, c := range changes {
		if maxHpBuffStatTypes[c.Type()] {
			return true
		}
	}
	return false
}

// markBerserkDirtyOnMaxHpChange schedules a grace-deferred berserk
// re-evaluation when any change set affects max HP. This service is the
// producer of the buff event atlas-effective-stats consumes to recompute max
// HP, so an immediate re-evaluation would read the stale value (design D5).
// Untracked characters are no-ops inside the berserk registry.
func markBerserkDirtyOnMaxHpChange(l logrus.FieldLogger, ctx context.Context, characterId uint32, changeSets ...[]stat.Model) {
	for _, cs := range changeSets {
		if affectsMaxHp(cs) {
			if err := berserk.NewProcessor(l, ctx).MarkMaxHpDirty(characterId); err != nil {
				l.WithError(err).Warnf("Unable to mark berserk dirty for character [%d].", characterId)
			}
			return
		}
	}
}
