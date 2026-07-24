package tv

import (
	"context"

	"github.com/sirupsen/logrus"

	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-packet/tv/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

// TvMessageType is the SEMANTIC message style a domain service selects when
// arming the Maple TV UI. The concrete client-interpreted wire byte (design-
// cited from IDA v83/v95: 0 = normal, 1 = star, 2 = heart — Tasks 19-20
// IDA-verify per version) is config-resolved per tenant from the tenant
// TvSetMessage writer's "messageTypes" table — the wire value is never a Go
// literal (DOM-25).
type TvMessageType string

const (
	TvMessageNormal TvMessageType = "NORMAL" // seed 0
	TvMessageStar   TvMessageType = "STAR"   // seed 1
	TvMessageHeart  TvMessageType = "HEART"  // seed 2
)

// TvResultReason is the SEMANTIC notice a domain service selects when a TV
// message submission is rejected. The concrete client-interpreted wire byte
// is config-resolved per tenant from the tenant TvSendMessageResult writer's
// "errorCodes" table — the wire value is never a Go literal (DOM-25). All
// three reasons are declared here (they enumerate the client's switch arms,
// mirroring ViciousHammerFailureReason); only QUEUE_TOO_LONG has a call site
// today. Seed values below are IDA-verified for v83
// (CMapleTVMan::OnSendMessageResult@0x6373a0, task-123 phase 19) — the
// original Cosmic-derived guess had WRONG_USER/QUEUE_TOO_LONG swapped; the
// gms_v83 seed template has been corrected to match.
type TvResultReason string

const (
	TvResultGmMessage    TvResultReason = "GM_MESSAGE"     // seed 1 (IDA-verified v83)
	TvResultWrongUser    TvResultReason = "WRONG_USER"     // seed 2 (IDA-verified v83)
	TvResultQueueTooLong TvResultReason = "QUEUE_TOO_LONG" // seed 3 (IDA-verified v83)
)

// TvSetMessageBody arms the Maple TV UI. messageType is resolved from the
// tenant writer's "messageTypes" table; every other field is codec-internal
// (the flag byte is computed by the constructor from whether receiverLook is
// non-nil — A1.4, exempt from resolution). An unconfigured msgType resolves
// to 99, which the client renders as a generic "Unknown error" notice.
func TvSetMessageBody(msgType TvMessageType, senderLook model.Avatar, senderName string, receiverName string, lines [5]string, totalWaitSeconds uint32, receiverLook *model.Avatar) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("messageTypes", string(msgType), func(messageType byte) packet.Encoder {
		return clientbound.NewTvSetMessage(messageType, senderLook, senderName, receiverName, lines, totalWaitSeconds, receiverLook)
	})
}

// TvClearMessageBody tears down the Maple TV UI. The body is empty — no
// resolution needed.
func TvClearMessageBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return clientbound.NewTvClearMessage().Encode
}

// TvSendMessageResultSuccessBody reports a successful TV message submission
// (00). No resolution needed.
func TvSendMessageResultSuccessBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return clientbound.NewTvSendMessageResultSuccess().Encode
}

// TvSendMessageResultErrorBody reports a rejected TV message submission. The
// notice selector is resolved from the tenant writer's "errorCodes" table. An
// unconfigured reason resolves to 99, which the client renders as a generic
// "Unknown error" notice.
func TvSendMessageResultErrorBody(reason TvResultReason) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("errorCodes", string(reason), func(code byte) packet.Encoder {
		return clientbound.NewTvSendMessageResultError(code)
	})
}
