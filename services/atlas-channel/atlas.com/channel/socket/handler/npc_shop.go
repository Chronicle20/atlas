package handler

import (
	"atlas-channel/npc/shops"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	npc2 "github.com/Chronicle20/atlas/libs/atlas-packet/npc/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const (
	NPCShopOperationBuy      = "BUY"
	NPCShopOperationSell     = "SELL"
	NPCShopOperationRecharge = "RECHARGE"
	NPCShopOperationLeave    = "LEAVE"
)

func NPCShopHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	sp := shops.NewProcessor(l, ctx)
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		pk := npc2.Shop{}
		pk.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", pk.Operation(), pk.String())
		op := pk.Op()
		if isNPCShopOperation(l)(readerOptions, op, NPCShopOperationBuy) {
			sb := &npc2.ShopBuy{}
			sb.Decode(l, ctx)(r, readerOptions)
			err := sp.BuyItem(s.CharacterId(), sb.Slot(), sb.ItemId(), uint32(sb.Quantity()), sb.DiscountPrice())
			if err != nil {
				l.WithError(err).Errorf("Failed to send shop buy command for character [%d].", s.CharacterId())
			}
			return
		}
		if isNPCShopOperation(l)(readerOptions, op, NPCShopOperationSell) {
			ss := &npc2.ShopSell{}
			ss.Decode(l, ctx)(r, readerOptions)
			err := sp.SellItem(s.CharacterId(), ss.Slot(), ss.ItemId(), uint32(ss.Quantity()))
			if err != nil {
				l.WithError(err).Errorf("Failed to send shop sell command for character [%d].", s.CharacterId())
			}
			return
		}
		if isNPCShopOperation(l)(readerOptions, op, NPCShopOperationRecharge) {
			sr := &npc2.ShopRecharge{}
			sr.Decode(l, ctx)(r, readerOptions)
			err := sp.RechargeItem(s.CharacterId(), sr.Slot())
			if err != nil {
				l.WithError(err).Errorf("Failed to send shop recharge command for character [%d].", s.CharacterId())
			}
			return
		}
		if isNPCShopOperation(l)(readerOptions, op, NPCShopOperationLeave) {
			err := sp.ExitShop(s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Failed to send shop exit command for character [%d].", s.CharacterId())
			}
			return
		}
		l.Warnf("Character [%d] issued unhandled npc shop operation with operation [%d].", s.CharacterId(), op)
	}
}

func isNPCShopOperation(l logrus.FieldLogger) func(options map[string]interface{}, op byte, key string) bool {
	return func(options map[string]interface{}, op byte, key string) bool {
		var genericCodes interface{}
		var ok bool
		if genericCodes, ok = options["operations"]; !ok {
			l.Errorf("Code [%s] not configured for use.", key)
			return false
		}

		var codes map[string]interface{}
		if codes, ok = genericCodes.(map[string]interface{}); !ok {
			l.Errorf("Code [%s] not configured for use.", key)
			return false
		}

		res, ok := codes[key].(float64)
		if !ok {
			l.Errorf("Code [%s] not configured for use.", key)
			return false
		}
		return byte(res) == op
	}
}
