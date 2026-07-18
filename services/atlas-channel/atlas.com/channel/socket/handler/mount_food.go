package handler

import (
	"atlas-channel/food"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	mountsb "github.com/Chronicle20/atlas/libs/atlas-packet/mount/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// MountFoodHandleFunc handles the serverbound taming-mob (mount) food packet
// (opcode 0x4D, SendTamingMobFoodItemUseRequest). It performs no item mutation;
// it forwards a feed command to consumables, which decrements the item and
// validates the classification-226 gate (Task 32). worldId flows via the field.
func MountFoodHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := mountsb.Food{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		_ = food.NewProcessor(l, ctx).RequestFeed(s.Field(), character.Id(s.CharacterId()), p.Slot(), p.ItemId())
	}
}
