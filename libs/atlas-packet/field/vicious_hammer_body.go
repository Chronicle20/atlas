package field

import (
	"context"

	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	"github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

type ViciousHammerMode string

const (
	ViciousHammerModeOpen    ViciousHammerMode = "OPEN"
	ViciousHammerModeSuccess ViciousHammerMode = "SUCCESS"
	ViciousHammerModeFailure ViciousHammerMode = "FAILURE"
)

// ViciousHammerFailureReason is the SEMANTIC notice a domain service selects
// for a failed hammer use. The concrete client-interpreted wire byte (1 = not
// upgradable, 2 = cap reached, 3 = Horntail, 0 = unknown on stock GMS) is
// config-resolved per tenant from the writer's "errorCodes" table — the wire
// value is never a Go literal (DOM-25).
type ViciousHammerFailureReason string

const (
	ViciousHammerReasonUnknown       ViciousHammerFailureReason = "UNKNOWN"
	ViciousHammerReasonNotUpgradable ViciousHammerFailureReason = "NOT_UPGRADABLE"
	ViciousHammerReasonCapReached    ViciousHammerFailureReason = "CAP_REACHED"
	ViciousHammerReasonHorntail      ViciousHammerFailureReason = "HORNTAIL"
)

// ViciousHammerOpenBody arms the CUIItemUpgrade gauge. token is the
// server-chosen round-trip value the client echoes in ITEM_UPGRADE_UPDATE;
// hammerCount is the target equip's hammersApplied AFTER this use (the client
// reuses it for the terminal "2 - count upgrades are left" success notice).
func ViciousHammerOpenBody(token uint32, hammerCount uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(ViciousHammerModeOpen), func(mode byte) packet.Encoder {
		return clientbound.NewViciousHammerOpen(mode, token, hammerCount)
	})
}

// ViciousHammerSuccessBody closes the dialog with the success notice. The
// client treats any non-zero flag as "Unknown error %d", so the flag is
// fixed to 0.
func ViciousHammerSuccessBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(ViciousHammerModeSuccess), func(mode byte) packet.Encoder {
		return clientbound.NewViciousHammerSuccess(mode, 0)
	})
}

// ViciousHammerFailureBody closes the dialog with the notice selected by
// reason. BOTH the dispatcher mode byte ("operations"/FAILURE) and the notice
// selector ("errorCodes"/<reason>) are resolved from the tenant writer config
// — neither is hard-coded (DOM-25). An unconfigured reason resolves to 99,
// which the client renders as a generic "Unknown error" notice.
func ViciousHammerFailureBody(reason ViciousHammerFailureReason) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := atlas_packet.ResolveCode(l, options, "operations", string(ViciousHammerModeFailure))
			code := atlas_packet.ResolveCode(l, options, "errorCodes", string(reason))
			return clientbound.NewViciousHammerFailure(mode, uint32(code)).Encode(l, ctx)(options)
		}
	}
}
