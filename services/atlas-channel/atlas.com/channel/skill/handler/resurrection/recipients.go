package resurrection

import (
	"context"

	"atlas-channel/data/skill/effect"
	channelhandler "atlas-channel/skill/handler"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/sirupsen/logrus"
)

// selectDeadParty / selectDeadMap are seams (aliases to the shared dead-target
// selectors) so the variant dispatch is unit-testable without the live stack.
var selectDeadParty = channelhandler.SelectDeadInRangePartyMembers
var selectDeadMap = channelhandler.SelectDeadInRangeMapPlayers

// selectByVariant routes each Resurrection variant to its recipient selector:
// Bishop -> dead party members in range; GM / SuperGM -> all dead players in
// range (party-agnostic).
func selectByVariant(
	l logrus.FieldLogger, ctx context.Context,
	f field.Model, casterId uint32, casterX, casterY int16,
	e effect.Model, memberBitmap byte, skillId skill2.Id,
) []channelhandler.PartyRecipient {
	switch skillId {
	case skill2.BishopResurrectionId:
		return selectDeadParty(l, ctx, f, casterId, casterX, casterY, e, memberBitmap)
	default:
		// GmResurrectionId / SuperGmResurrectionId — party-agnostic.
		return selectDeadMap(l, ctx, f, casterId, casterX, casterY, e)
	}
}
