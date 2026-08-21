package parcel

// EnvStatusEventTopic names the parcel status event topic — the arrival
// notification channel to the rest of the platform. It is a sibling of
// custody's EnvStatusTopic (kafka/message/custody/kafka.go), not the same
// topic: custody acks saga steps, this one notifies players.
const EnvStatusEventTopic = "EVENT_TOPIC_PARCEL_STATUS"

// StatusEventParcelArrived notifies a parcel's RECIPIENT that a parcel has
// become receivable (design §7.1 — no notification tier ladder, one arrival
// event).
const StatusEventParcelArrived = "PARCEL_ARRIVED"

// StatusEventParcelSent notifies a parcel's SENDER that their parcel_send
// saga completed, so the channel can announce PARCEL[SUCCESSFULLY_SENT] and
// the client re-enables its send tab. Emitted from handleAcceptToParcel:
// accept_to_parcel is the last step of parcel_send (award_mesos → optional
// ticket destroy → transfer_to_parcel → release_from_character +
// accept_to_parcel), so the row create IS the completion.
const StatusEventParcelSent = "PARCEL_SENT"

// StatusEvent is the generic parcel status event envelope, addressed by
// CharacterId to whichever party the event concerns — the recipient for
// PARCEL_ARRIVED, the sender for PARCEL_SENT. It mirrors
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

// StatusEventParcelSentBody carries nothing: PARCEL[SUCCESSFULLY_SENT] is a
// bare mode byte (design §5.2, 0x12 — SP_3901 plus the send-tab reset), and
// the addressee is already the envelope's CharacterId.
type StatusEventParcelSentBody struct{}
