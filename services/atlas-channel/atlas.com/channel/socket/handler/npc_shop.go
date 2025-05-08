package handler

import (
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

func NPCShopHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		op := r.ReadByte()
		if isNPCShopOperation(l)(readerOptions, op, NPCShopOperationBuy) {
			slot := r.ReadUint16()
			itemId := r.ReadUint32()
			quantity := r.ReadUint16()
			discountPrice := r.ReadUint32()
			l.Debugf("Character [%d] has requested to buy [%d] item [%d] from slot [%d] in NPC shop at price [%d].", s.CharacterId(), quantity, itemId, slot, discountPrice)
			return
		}
		if isNPCShopOperation(l)(readerOptions, op, NPCShopOperationSell) {
			slot := r.ReadUint16()
			itemId := r.ReadUint32()
			quantity := r.ReadUint16()
			l.Debugf("Character [%d] has requested to sell [%d] item [%d] from slot [%d] in NPC shop.", s.CharacterId(), quantity, itemId, slot)
			return
		}
		if isNPCShopOperation(l)(readerOptions, op, NPCShopOperationRecharge) {
			slot := r.ReadUint16()
			l.Debugf("Character [%d] has requested to recharge slot [%d] in NPC shop.", s.CharacterId(), slot)
			return
		}
		if isNPCShopOperation(l)(readerOptions, op, NPCShopOperationLeave) {
			l.Debugf("Character [%d] has left NPC shop.", s.CharacterId())
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
