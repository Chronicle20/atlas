package writer

import (
	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	reportcb "github.com/Chronicle20/atlas/libs/atlas-packet/report/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

// ClaimResultCode keys the ClaimResult writer's tenant operations table
// (DOM-25). The full verified mode set lives in the table; REPORTED_NOTICE,
// CANNOT_CONNECT, TIME_WINDOW and FALSE_REPORT_CITED are expressible but
// unused — nothing in Atlas produces those conditions.
type ClaimResultCode string

const (
	ClaimResultSuccessCode    ClaimResultCode = "SUCCESS"
	ClaimResultTryAgain       ClaimResultCode = "TRY_AGAIN"
	ClaimResultRecheckName    ClaimResultCode = "RECHECK_NAME"
	ClaimResultNotEnoughMesos ClaimResultCode = "NOT_ENOUGH_MESOS"
	ClaimResultExceeded       ClaimResultCode = "EXCEEDED"
)

func ClaimResultSuccessBody(hasRemaining bool, remaining int32) packet.Encode {
	return atlas_packet.WithResolvedCode("operations", string(ClaimResultSuccessCode), func(code byte) packet.Encoder {
		return reportcb.NewClaimResultSuccess(code, hasRemaining, remaining)
	})
}

func ClaimResultNoticeBody(key ClaimResultCode) packet.Encode {
	return atlas_packet.WithResolvedCode("operations", string(key), func(code byte) packet.Encoder {
		return reportcb.NewClaimResultNotice(code)
	})
}
