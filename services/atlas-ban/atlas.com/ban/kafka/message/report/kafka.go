package report

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	EnvCommandTopic   = "COMMAND_TOPIC_REPORT"
	CommandTypeCreate = "CREATE"

	EnvEventTopicStatus = "EVENT_TOPIC_REPORT_STATUS"
	EventStatusCreated  = "CREATED"
	EventStatusError    = "ERROR"

	ErrorCodeNotFound = "NOT_FOUND"
	ErrorCodeInternal = "INTERNAL"

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

type StatusEvent struct {
	ReportId   uuid.UUID `json:"reportId"` // uuid.Nil on ERROR
	Kind       string    `json:"kind"`
	WorldId    world.Id  `json:"worldId"`
	ReporterId uint32    `json:"reporterId"`
	Status     string    `json:"status"`    // CREATED | ERROR
	ErrorCode  string    `json:"errorCode"` // NOT_FOUND | INTERNAL; empty on CREATED
}
