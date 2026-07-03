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

// ViciousHammerOpenBody arms the CUIItemUpgrade gauge. token is the
// server-chosen round-trip value the client echoes in ITEM_UPGRADE_UPDATE;
// hammerCount is the target equip's current hammersApplied.
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
// errorCode (1 = not upgradable, 2 = cap reached, 3 = Horntail Necklace,
// other = unknown error).
func ViciousHammerFailureBody(errorCode uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", string(ViciousHammerModeFailure), func(mode byte) packet.Encoder {
		return clientbound.NewViciousHammerFailure(mode, errorCode)
	})
}
