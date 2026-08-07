package writer

import (
	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	reportcb "github.com/Chronicle20/atlas/libs/atlas-packet/report/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

// SueResultCode keys the SueCharacterResult writer's tenant operations
// table (DOM-25). DAILY_LIMIT and REPORTED_NOTICE exist in the table but
// have no v1 emitter (quota enforcement and accused notification deferred).
type SueResultCode string

const (
	SueResultSuccess        SueResultCode = "SUCCESS"
	SueResultUnableToLocate SueResultCode = "UNABLE_TO_LOCATE"
	SueResultGenericFailure SueResultCode = "GENERIC_FAILURE"
)

func SueCharacterResultBody(key SueResultCode) packet.Encode {
	return atlas_packet.WithResolvedCode("operations", string(key), func(code byte) packet.Encoder {
		return reportcb.NewSueCharacterResult(code)
	})
}
