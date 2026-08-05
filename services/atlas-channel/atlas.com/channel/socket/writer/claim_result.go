package writer

import (
	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	reportcb "github.com/Chronicle20/atlas/libs/atlas-packet/report/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

// ClaimResultCode keys the ClaimResult writer's tenant operations table
// (DOM-25). The full verified mode set lives in the table; v1 emits only
// SUCCESS / TRY_AGAIN / RECHECK_NAME.
type ClaimResultCode string

const (
	ClaimResultSuccessCode ClaimResultCode = "SUCCESS"
	ClaimResultTryAgain    ClaimResultCode = "TRY_AGAIN"
	ClaimResultRecheckName ClaimResultCode = "RECHECK_NAME"
)

// ClaimResultSuccessRemaining is the v1 display-only weekly-report count
// passed as `remaining` to ClaimResultSuccessBody on a successful claim. It
// is not enforced server-side in v1 (quota enforcement deferred); the
// client only renders it. Exported for the status consumer (Task 16) and
// session bootstrap (Task 17) callers to reference rather than inlining 100.
const ClaimResultSuccessRemaining int32 = 100

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
