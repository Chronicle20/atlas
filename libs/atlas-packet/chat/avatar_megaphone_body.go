package chat

import (
	"context"

	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	"github.com/Chronicle20/atlas/libs/atlas-packet/chat/clientbound"
	"github.com/sirupsen/logrus"
)

// AvatarMegaphoneResultReason is the SEMANTIC notice a domain service selects
// when an avatar megaphone use cannot proceed. The concrete client-interpreted
// wire byte (design-cited from IDA v83/v95: 83 = waiting line, 84 = level
// gate — Tasks 19-20 IDA-verify per version) is config-resolved per tenant
// from the writer's "errorCodes" table — the wire value is never a Go literal
// (DOM-25).
type AvatarMegaphoneResultReason string

const (
	AvatarMegaphoneWaitingLine AvatarMegaphoneResultReason = "WAITING_LINE" // seed 83
	AvatarMegaphoneLevelGate   AvatarMegaphoneResultReason = "LEVEL_GATE"   // seed 84
)

// AvatarMegaphoneResultBody resolves the notice selector for reason from the
// tenant writer's "errorCodes" table and emits the code-only AvatarMegaphoneResult
// wire (no trailing message: neither WAITING_LINE nor LEVEL_GATE carries one).
// An unconfigured reason resolves to 99, which the client renders as a generic
// "Unknown error" notice.
func AvatarMegaphoneResultBody(reason AvatarMegaphoneResultReason) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			code := atlas_packet.ResolveCode(l, options, "errorCodes", string(reason))
			return clientbound.NewAvatarMegaphoneResult(code, "").Encode(l, ctx)(options)
		}
	}
}
