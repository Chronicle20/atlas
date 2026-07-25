package handler

import (
	"atlas-channel/merchant"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	merchantpkt "github.com/Chronicle20/atlas/libs/atlas-packet/merchant"
	merchantcb "github.com/Chronicle20/atlas/libs/atlas-packet/merchant/clientbound"
	merchantsb "github.com/Chronicle20/atlas/libs/atlas-packet/merchant/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// HiredMerchantOperationHandleFunc handles the entrusted-shop (hired-merchant)
// serverbound dispatcher. The only mode the client emits is
// ModeEntrustedShopCheck, sent when a player uses a hired-merchant permit (a
// cash-shop slot item). The server MUST reply with an
// ENTRUSTED_SHOP_CHECK_RESULT — the client blocks on it
// (CWvsContext::OnEntrustedShopCheckResult, v83 @0xa27d75):
//
//	mode OPEN_SHOP (7, no body)             -> client fires SendOpenShopRequest,
//	                                           opening the store-create dialog.
//	mode ERROR_UNKNOWN (8, int + channel)   -> "your store is currently open in
//	                                           channel %s FM %d" notice (int%100
//	                                           is the displayed FM room @0xa27e6c).
//	mode ERROR_RETRIEVE_FROM_FREDRICK (9)   -> "retrieve items from Fredrick" notice.
//	mode ERROR_UNABLE_TO_OPEN_THE_STORE (11)-> generic unable notice.
//
// Without a reply the permit is a silent no-op. The in-shop lifecycle (put
// item, buy, exit, withdraw meso, blacklist, name change, ...) arrives via
// CharacterInteraction and is handled in character_interaction.go, not here.
func HiredMerchantOperationHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
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

		announce := session.Announce(l)(ctx)(wp)(merchantcb.HiredMerchantOperationWriter)

		// A character may only operate a single hired merchant at a time. Only
		// non-Closed hired-merchant rows count — a Closed history row must not
		// poison the check forever, and personal shops are a separate slot.
		mp := merchant.NewProcessor(l, ctx)
		shops, err := mp.GetByCharacterId(s.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to query existing shops for character [%d].", s.CharacterId())
			_ = announce(merchantpkt.HiredMerchantOperationErrorUnableToOpenTheStoreBody())(s)
			return
		}
		for _, sh := range shops {
			if sh.ShopType() != merchant.HiredMerchantShopType || sh.State() == merchant.StateClosed {
				continue
			}
			if sh.State() == merchant.StateOpen || sh.State() == merchant.StateMaintenance {
				// Faithful notice: the store is already running — the client
				// renders mapId%100 as the FM room number and resolves the
				// channel name from the byte.
				l.Debugf("Character [%d] already operates hired merchant [%s]; sending already-open notice.", s.CharacterId(), sh.Id())
				_ = announce(merchantpkt.HiredMerchantOperationErrorUnknownBody(sh.MapId(), byte(sh.ChannelId())))(s)
				return
			}
			// A lingering Draft (setup session) blocks with the generic notice.
			l.Debugf("Character [%d] has hired merchant [%s] in state [%d]; refusing permit check.", s.CharacterId(), sh.Id(), sh.State())
			_ = announce(merchantpkt.HiredMerchantOperationErrorUnableToOpenTheStoreBody())(s)
			return
		}

		// Unclaimed Fredrick items/mesos must be retrieved before opening anew.
		hasPending, err := mp.HasFrederickPending(s.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to query Fredrick status for character [%d].", s.CharacterId())
			_ = announce(merchantpkt.HiredMerchantOperationErrorUnableToOpenTheStoreBody())(s)
			return
		}
		if hasPending {
			l.Debugf("Character [%d] has pending Fredrick items; refusing permit check.", s.CharacterId())
			_ = announce(merchantpkt.HiredMerchantOperationErrorRetrieveFromFredrickBody())(s)
			return
		}

		// Check passed: mode OPEN_SHOP tells the client to open the create
		// dialog; the actual shop is created when the client follows up with
		// CharacterInteraction CREATE -> PlaceShop.
		l.Debugf("Character [%d] is permitted to open a hired merchant.", s.CharacterId())
		_ = announce(merchantpkt.HiredMerchantOperationOpenShopBody())(s)
	}
}
