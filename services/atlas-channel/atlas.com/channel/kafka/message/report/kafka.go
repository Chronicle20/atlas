package report

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvCommandTopic topic.Token = "COMMAND_TOPIC_REPORT"
)

const (
	CommandTypeCreate = "CREATE"
)

const (
	EnvEventTopicStatus topic.Token = "EVENT_TOPIC_REPORT_STATUS"
)

const (
	EventStatusCreated = "CREATED"
	EventStatusError   = "ERROR"

	ErrorCodeNotFound      = "NOT_FOUND"
	ErrorCodeInternal      = "INTERNAL"
	ErrorCodeQuotaExceeded = "QUOTA_EXCEEDED"

	KindSue   = "sue"
	KindClaim = "claim"
)

type Command[E any] struct {
	Type string `json:"type"`
	Body E      `json:"body"`
}

// CreateCommandBody carries the report exactly as supplied on the wire.
// Accused identity is mechanism-dependent: claim and v95 sue supply
// AccusedName (v95 sue's sub-command string is treated as the target name);
// legacy sue (v83/v84/v87) supplies AccusedId. The consumer resolves the
// missing half via atlas-character and rejects unresolvable targets.
type CreateCommandBody struct {
	Kind        string     `json:"kind"`
	WorldId     world.Id   `json:"worldId"`
	ChannelId   channel.Id `json:"channelId"`
	ReporterId  uint32     `json:"reporterId"`
	AccusedId   uint32     `json:"accusedId"`
	AccusedName string     `json:"accusedName"`
	ReasonType  byte       `json:"reasonType"`
	Description string     `json:"description"`
	ChatClaim   bool       `json:"chatClaim"`
	ChatLog     string     `json:"chatLog"`
}

// StatusEvent reports the outcome of a create command back to the channel
// that submitted it. HasRemaining/Remaining carry the reporter's claim quota
// standing as atlas-ban computed it, and are meaningful only on a claim
// CREATED — they go straight into CLAIM_RESULT's success body. Sue leaves both
// zero-valued.
type StatusEvent struct {
	ReportId     uuid.UUID `json:"reportId"` // uuid.Nil on ERROR
	Kind         string    `json:"kind"`
	WorldId      world.Id  `json:"worldId"`
	ReporterId   uint32    `json:"reporterId"`
	Status       string    `json:"status"`    // CREATED | ERROR
	ErrorCode    string    `json:"errorCode"` // NOT_FOUND | INTERNAL | QUOTA_EXCEEDED; empty on CREATED
	HasRemaining bool      `json:"hasRemaining"`
	Remaining    int32     `json:"remaining"`
}
