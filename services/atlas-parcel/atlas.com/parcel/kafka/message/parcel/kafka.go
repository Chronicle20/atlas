package parcel

// EnvStatusEventTopic names the parcel status event topic — the arrival
// notification channel to the rest of the platform. It is a sibling of
// custody's EnvStatusTopic (kafka/message/custody/kafka.go), not the same
// topic: custody acks saga steps, this one notifies players.
const EnvStatusEventTopic = "EVENT_TOPIC_PARCEL_STATUS"

// StatusEventParcelArrived is the only status event this topic carries
// today (design §7.1 — no notification tier ladder, one arrival event).
const StatusEventParcelArrived = "PARCEL_ARRIVED"

// StatusEvent is the generic parcel status event envelope, addressed to the
// parcel's recipient by CharacterId — mirrors
// services/atlas-merchant/atlas.com/merchant/kafka/message/merchant/kafka.go's
// StatusEvent[E], the shape the channel-side handler expects
// (IfPresentByCharacterId keyed off CharacterId, task-25).
type StatusEvent[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

// StatusEventParcelArrivedBody carries what the channel-side alarm needs to
// announce ALARM_NAMED (task-25, design §7.1): the sender's display name and
// whether the parcel holds an item. There is no notification tier here — a
// parcel has one arrival event, not Frederick's escalating rot ladder.
type StatusEventParcelArrivedBody struct {
	SenderName string `json:"senderName"`
	HasItem    bool   `json:"hasItem"`
}
