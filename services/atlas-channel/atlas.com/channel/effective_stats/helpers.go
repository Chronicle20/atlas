package effective_stats

import (
	"context"
	"math"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/sirupsen/logrus"
)

// MaxHpOrBase narrows the effective MaxHp from atlas-effective-stats into
// the uint16 range used by client packets, falling back to the character's
// base MaxHp when the upstream returned zero or out-of-range. Mirrors the
// defensive strategy in atlas-character's resolveEffectiveMax.
func MaxHpOrBase(effective uint32, base uint16) uint16 {
	if effective == 0 {
		return base
	}
	if effective > math.MaxUint16 {
		return math.MaxUint16
	}
	return uint16(effective)
}

// MaxMpOrBase is the MP counterpart to MaxHpOrBase.
func MaxMpOrBase(effective uint32, base uint16) uint16 {
	if effective == 0 {
		return base
	}
	if effective > math.MaxUint16 {
		return math.MaxUint16
	}
	return uint16(effective)
}

// ResolveCharacterMaxes fetches effective HP/MP caps for the character via
// atlas-effective-stats and returns values safe to send to the v83 client
// (gear/buff/passive bonuses already applied). On fetch error or a zero
// upstream value, falls back to the character's base maxes so a stats-service
// regression doesn't clamp the client display to zero.
func ResolveCharacterMaxes(l logrus.FieldLogger, ctx context.Context, worldId world.Id, channelId channel.Id, characterId uint32, baseMaxHp, baseMaxMp uint16) (uint16, uint16) {
	stats, err := NewProcessor(l, ctx).GetByCharacterId(worldId, channelId, characterId)
	if err != nil {
		l.WithError(err).Debugf("ResolveCharacterMaxes: effective stats fetch failed for character [%d]; using base MaxHp=[%d] MaxMp=[%d].", characterId, baseMaxHp, baseMaxMp)
		return baseMaxHp, baseMaxMp
	}
	return MaxHpOrBase(stats.MaxHp, baseMaxHp), MaxMpOrBase(stats.MaxMp, baseMaxMp)
}
