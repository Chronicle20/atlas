package parcel

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

// COMMAND_TOPIC_PARCEL carries commands dispatched to atlas-channel that open
// the Duey parcel dialog for a character (task-241 Task 19's producer,
// services/atlas-saga-orchestrator/.../kafka/message/parcel/kafka.go —
// ShowParcelCommand there is this same shape).
const (
	EnvCommandTopic       topic.Token = "COMMAND_TOPIC_PARCEL"
	CommandTypeShowParcel             = "SHOW_PARCEL"
)

// ShowParcelCommand is received from the saga-orchestrator to display the
// Duey parcel dialog. Quick discriminates the two entry points: false is the
// NPC path (PARCEL[OPEN] with the mailbox), true is the Quick Delivery
// Ticket path (PARCEL[OPEN_QUICK], no list — task-241 design §5.2, §9.5).
type ShowParcelCommand struct {
	TransactionId uuid.UUID  `json:"transactionId"`
	WorldId       world.Id   `json:"worldId"`
	ChannelId     channel.Id `json:"channelId"`
	CharacterId   uint32     `json:"characterId"`
	NpcId         uint32     `json:"npcId"`
	Quick         bool       `json:"quick"`
	Type          string     `json:"type"`
}

// EnvStatusEventTopic names the parcel status event topic — the arrival
// notification channel to the rest of the platform. It is a sibling of
// custody's EnvStatusTopic, not the same topic: custody acks saga steps,
// this one notifies players. Mirrors atlas-parcel's producer-side
// kafka/message/parcel/kafka.go (task-241 Task 24) — field names must match
// exactly since these are separate Go modules.
const EnvStatusEventTopic topic.Token = "EVENT_TOPIC_PARCEL_STATUS"

// StatusEventParcelArrived notifies a parcel's RECIPIENT that a parcel has
// become receivable (design §7.1 — no notification tier ladder, one arrival
// event).
const StatusEventParcelArrived = "PARCEL_ARRIVED"

// StatusEventParcelSent notifies a parcel's SENDER that their parcel_send
// saga completed — atlas-parcel emits it from handleAcceptToParcel, the
// saga's last step. The channel answers with PARCEL[SUCCESSFULLY_SENT].
const StatusEventParcelSent = "PARCEL_SENT"

// StatusEventParcelReceived notifies a parcel's RECIPIENT that
// atlas-parcel's handleReleaseFromParcel completed — the row transitioned to
// received. The channel answers with PARCEL[PARCEL_REMOVED] (kind Claimed),
// which both removes the row and re-enables the dialog (v83 IDB @0x6f56ea,
// case 23: it calls RemoveParcel then SetCtrlEnabled(1) itself, so no
// separate unlock packet is needed).
const StatusEventParcelReceived = "PARCEL_RECEIVED"

// StatusEvent is the generic parcel status event envelope, addressed to the
// parcel's recipient by CharacterId — mirrors atlas-parcel's producer-side
// StatusEvent[E] (task-241 Task 24) and
// services/atlas-merchant/atlas.com/merchant/kafka/message/merchant/kafka.go's
// StatusEvent[E], the shape handleParcelArrivedEvent expects
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

// StatusEventParcelSentBody carries nothing — PARCEL[SUCCESSFULLY_SENT] is a
// bare mode byte (design §5.2, 0x12) and the addressee is the envelope's
// CharacterId. Mirrors atlas-parcel's producer-side body; field-for-field
// identity matters because these are separate Go modules.
type StatusEventParcelSentBody struct{}

// StatusEventParcelReceivedBody carries the released parcel's id so
// handleParcelReceivedEvent can project it onto the wire's uint32 parcelId
// (dueyparcel.WireId) for PARCEL[PARCEL_REMOVED]. Mirrors atlas-parcel's
// producer-side body; field-for-field identity matters because these are
// separate Go modules.
type StatusEventParcelReceivedBody struct {
	ParcelId uuid.UUID `json:"parcelId"`
}
