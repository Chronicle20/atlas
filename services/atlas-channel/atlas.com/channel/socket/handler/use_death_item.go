package handler

import (
	"atlas-channel/character"
	channelInventory "atlas-channel/inventory"
	_map "atlas-channel/map"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	charcb "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	charsb "github.com/Chronicle20/atlas/libs/atlas-packet/character/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// canShowTombEffect authorises the relay. The client hard-codes the Wheel of
// Destiny id at the send site and only builds the revive dialog while dead, so
// anything else is a forged or stale request and is dropped silently on the
// wire (logged server-side).
func canShowTombEffect(hp uint16, itemId uint32, wheelQuantity uint32) bool {
	if hp != 0 {
		return false
	}
	if !item.IsWheelOfFortune(item.Id(itemId)) {
		return false
	}
	return wheelQuantity >= 1
}

// CharacterUseDeathItemHandleFunc relays CUIRevive's tomb-effect request to the
// other players in the map. It deliberately consumes nothing and changes no
// state: the wheel is spent by the MAP_CHANGE revive path (respawn.Respawn),
// which the client sends separately when the player actually presses OK.
func CharacterUseDeathItemHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := charsb.UseDeathItem{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		c, err := character.NewProcessor(l, ctx).GetById()(s.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to get character [%d] for death item tomb effect.", s.CharacterId())
			return
		}

		var quantity uint32
		inv, err := channelInventory.NewProcessor(l, ctx).GetByCharacterId(s.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to get inventory for character [%d] for death item tomb effect.", s.CharacterId())
			return
		}
		if a, found := inv.Cash().FindFirstByItemId(uint32(item.WheelOfFortuneId)); found && a != nil {
			quantity = a.Quantity()
		}

		if !canShowTombEffect(c.Hp(), p.ItemId(), quantity) {
			l.Warnf("Character [%d] requested a death item tomb effect for item [%d] with hp [%d] and [%d] charges. Ignoring.", s.CharacterId(), p.ItemId(), c.Hp(), quantity)
			return
		}

		// Owner excluded: CUserLocal::RequestUpgradeTombEffect already ran
		// CUser::ShowUpgradeTombEffect locally before sending, so echoing to
		// the sender would play the effect twice on their screen.
		err = _map.NewProcessor(l, ctx).ForOtherSessionsInMap(s.Field(), s.CharacterId(),
			session.Announce(l)(ctx)(wp)(charcb.CharacterShowUpgradeTombEffectWriter)(
				charcb.NewShowUpgradeTombEffect(s.CharacterId(), p.ItemId(), p.X(), p.Y()).Encode))
		if err != nil {
			l.WithError(err).Errorf("Unable to broadcast death item tomb effect for character [%d].", s.CharacterId())
		}
	}
}
