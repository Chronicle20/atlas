package handler

import (
	"atlas-channel/npc/shops"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const (
	NPCShopHandle            = "NPCShopHandle"
	NPCShopOperationBuy      = "BUY"
	NPCShopOperationSell     = "SELL"
	NPCShopOperationRecharge = "RECHARGE"
	NPCShopOperationLeave    = "LEAVE"
)

func NPCShopHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	sp := shops.NewProcessor(l, ctx)
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		op := r.ReadByte()
		if isNPCShopOperation(l)(readerOptions, op, NPCShopOperationBuy) {
			slot := r.ReadUint16()
			itemId := r.ReadUint32()
			quantity := r.ReadUint16()
			discountPrice := r.ReadUint32()
			err := sp.BuyItem(s.CharacterId(), slot, itemId, uint32(quantity), discountPrice)
			if err != nil {
				l.WithError(err).Errorf("Failed to send shop buy command for character [%d].", s.CharacterId())
			}
			return
		}
		if isNPCShopOperation(l)(readerOptions, op, NPCShopOperationSell) {
			slot := r.ReadInt16()
			itemId := r.ReadUint32()
			quantity := r.ReadUint16()
			err := sp.SellItem(s.CharacterId(), slot, itemId, uint32(quantity))
			if err != nil {
				l.WithError(err).Errorf("Failed to send shop sell command for character [%d].", s.CharacterId())
			}
			return
		}
		if isNPCShopOperation(l)(readerOptions, op, NPCShopOperationRecharge) {
			slot := r.ReadUint16()
			err := sp.RechargeItem(s.CharacterId(), slot)
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
