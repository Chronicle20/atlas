package handler

import (
	"atlas-channel/merchant"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	merchantsb "github.com/Chronicle20/atlas/libs/atlas-packet/merchant/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

// HiredMerchantOperationHandleFunc handles the entrusted-shop (hired-merchant)
// serverbound dispatcher (JMS v185 opcode 0x37). The only mode the client emits is
// ModeEntrustedShopCheck, sent when a player uses a hired-merchant permit (a cash-shop
// slot item). The in-shop lifecycle (put item, buy, exit, withdraw meso, blacklist,
// name change, ...) arrives via CharacterInteraction and is handled in
// character_interaction.go, not here.
func HiredMerchantOperationHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := merchantsb.Operation{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		mode := p.Mode()
		if mode != merchantsb.ModeEntrustedShopCheck {
			l.Debugf("Character [%d] issued unhandled hired-merchant operation mode [%d]; ignoring.", s.CharacterId(), mode)
			return
		}

		l.Debugf("Character [%d] requested hired-merchant permit check. cashItemSerialNumber [%d].", s.CharacterId(), p.CashItemSerialNumber())

		// A character may only operate a single hired merchant at a time. Verify with
		// the merchant service before allowing the entrusted-shop flow to proceed.
		mp := merchant.NewProcessor(l, ctx)
		shops, err := mp.GetByCharacterId(s.CharacterId())
		if err != nil {
			l.WithError(err).Debugf("Unable to query existing shops for character [%d]; treating as none.", s.CharacterId())
			return
		}
		if len(shops) > 0 {
			l.Debugf("Character [%d] already operates a shop; ignoring hired-merchant permit check.", s.CharacterId())
			return
		}

		// No existing shop: the permit check passes. The actual shop is created when the
		// client opens the entrusted-shop dialog (CharacterInteraction CREATE → PlaceShop).
		l.Debugf("Character [%d] is permitted to open a hired merchant.", s.CharacterId())
	}
}
