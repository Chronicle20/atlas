package resurrection

import (
	"atlas-channel/data/skill/effect"
	"context"

	channelhandler "atlas-channel/skill/handler"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

// selectDeadParty / selectDeadMap are seams (aliases to the shared dead-target
// selectors) so the variant dispatch is unit-testable without the live stack.
var (
	selectDeadParty = channelhandler.SelectDeadInRangePartyMembers
	selectDeadMap   = channelhandler.SelectDeadInRangeMapPlayers
)

// selectByVariant routes each Resurrection variant to its recipient selector:
// Bishop -> dead party members in range; GM / SuperGM -> all dead players in
// range (party-agnostic). skillId is the version-blind Identity the caller
// already resolved from the cast's wire id (task-187) -- NOT a raw wire id,
// since GmResurrection/SuperGmResurrection bind to different wire values
// across versions (e.g. wire 5101005 at v0.48 vs 9101005 at v0.83+).
func selectByVariant(
	l logrus.FieldLogger, ctx context.Context,
	f field.Model, casterId uint32, casterX, casterY int16,
	e effect.Model, memberBitmap byte, skillId skill2.Identity,
) []channelhandler.PartyRecipient {
	switch skillId {
	case skill2.BishopResurrection:
		return selectDeadParty(l, ctx, f, casterId, casterX, casterY, e, memberBitmap)
	default:
		// GmResurrection / SuperGmResurrection (and any unresolved id) —
		// party-agnostic.
		return selectDeadMap(l, ctx, f, casterId, casterX, casterY, e)
	}
}
