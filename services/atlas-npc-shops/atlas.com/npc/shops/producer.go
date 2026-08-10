package shops

import (
	"atlas-npc/kafka/message/shops"

	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func enteredEventProvider(characterId uint32, npcId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &shops.StatusEvent[shops.StatusEventEnteredBody]{
		CharacterId: characterId,
		Type:        shops.StatusEventTypeEntered,
		Body: shops.StatusEventEnteredBody{
			NpcTemplateId: npcId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func exitedEventProvider(characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &shops.StatusEvent[shops.StatusEventExitedBody]{
		CharacterId: characterId,
		Type:        shops.StatusEventTypeExited,
		Body:        shops.StatusEventExitedBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

// okEventProvider publishes the CONFIRM_SHOP_TRANSACTION OK arm, which the
// client requires to unlatch its shop dialog.
//
// The v83 client sets CShopDlg+0xFC to 1 the moment it sends a buy, sell or
// recharge request (CShopDlg::SendSellRequest @0x756a04,
// CShopDlg::SendRechargeRequest @0x756c28) and refuses to send another while
// that flag is set. CShopDlg::OnPacket @0x756da7 clears it — and it is the
// only thing that ever clears it — on receipt of this packet. A success that
// publishes no status event therefore leaves the dialog wedged until the
// player closes and reopens the shop, which constructs a fresh CShopDlg.
//
// The client wants exactly one status event per request: the same OnPacket
// throws CDisconnectException when the packet arrives with no request
// outstanding. Every terminal path through Buy/Sell/Recharge honours that, but
// nothing structurally enforces it — a redelivered command on the
// at-least-once COMMAND_TOPIC_NPC_SHOP re-runs the success path and publishes
// a second OK (the same redelivery already duplicated the item), and the
// `return err` paths that bail out mid-buffer publish none at all.
func okEventProvider(characterId uint32) model.Provider[[]kafka.Message] {
	return errorEventProvider(characterId, shops.ErrorOk)
}

func errorEventProvider(characterId uint32, errorMsg string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &shops.StatusEvent[shops.StatusEventErrorBody]{
		CharacterId: characterId,
		Type:        shops.StatusEventTypeError,
		Body: shops.StatusEventErrorBody{
			Error: errorMsg,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func levelLimitErrorEventProvider(characterId uint32, errorMsg string, levelLimit uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &shops.StatusEvent[shops.StatusEventErrorBody]{
		CharacterId: characterId,
		Type:        shops.StatusEventTypeError,
		Body: shops.StatusEventErrorBody{
			Error:      errorMsg,
			LevelLimit: levelLimit,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func reasonErrorEventProvider(characterId uint32, errorMsg string, reason string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &shops.StatusEvent[shops.StatusEventErrorBody]{
		CharacterId: characterId,
		Type:        shops.StatusEventTypeError,
		Body: shops.StatusEventErrorBody{
			Error:  errorMsg,
			Reason: reason,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
