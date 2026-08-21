package parcel

import (
	parcelmsg "atlas-parcel/kafka/message/parcel"

	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// ParcelArrivedStatusEventProvider builds a PARCEL_ARRIVED status event
// addressed to characterId — the parcel's recipient — carrying the sender's
// display name and whether the parcel holds an item. Keyed by characterId so
// every event for one recipient lands on the same partition in order,
// mirroring frederick's notificationProvider
// (services/atlas-merchant/atlas.com/merchant/frederick/notification_task.go).
func ParcelArrivedStatusEventProvider(characterId uint32, senderName string, hasItem bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &parcelmsg.StatusEvent[parcelmsg.StatusEventParcelArrivedBody]{
		CharacterId: characterId,
		Type:        parcelmsg.StatusEventParcelArrived,
		Body: parcelmsg.StatusEventParcelArrivedBody{
			SenderName: senderName,
			HasItem:    hasItem,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// ParcelSentStatusEventProvider builds a PARCEL_SENT status event addressed
// to characterId — the parcel's SENDER — so the channel can announce
// PARCEL[SUCCESSFULLY_SENT] and the client re-enables its send tab. Keyed by
// characterId for the same per-character ordering as the arrival event.
func ParcelSentStatusEventProvider(characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &parcelmsg.StatusEvent[parcelmsg.StatusEventParcelSentBody]{
		CharacterId: characterId,
		Type:        parcelmsg.StatusEventParcelSent,
		Body:        parcelmsg.StatusEventParcelSentBody{},
	}
	return producer.SingleMessageProvider(key, value)
}
