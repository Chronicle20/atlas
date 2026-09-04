package handler

import (
	"atlas-channel/character"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	charsb "github.com/Chronicle20/atlas/libs/atlas-packet/character/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// redeemStoredExperienceProcessorFunc is a test seam over the atlas-character
// redeem command emit (package-var injection precedent: karmaCharacterProcessorFunc
// in character_cash_item_use.go). Handler tests must not require a live Kafka
// broker to assert this handler reached RedeemStoredExperience.
var redeemStoredExperienceProcessorFunc = func(l logrus.FieldLogger, ctx context.Context) character.Processor {
	return character.NewProcessor(l, ctx)
}

// CharacterUseStoredExperienceHandleFunc handles USE_GACHA_EXP: the player
// clicked the EXP bar and confirmed charging the EXP banked by their Writs of
// Solomon. The request carries nothing but the tick — the client always
// redeems the whole balance — so every rule (zero balance, level > 50) is
// evaluated server-side in atlas-character.
func CharacterUseStoredExperienceHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := charsb.StoredExperienceUse{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		_ = redeemStoredExperienceProcessorFunc(l, ctx).RedeemStoredExperience(s.Field(), s.CharacterId())
	}
}
