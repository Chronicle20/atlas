package teleportrock

import (
	teleportrock2 "atlas-channel/kafka/message/teleportrock"
	"testing"

	trpkt "github.com/Chronicle20/atlas/libs/atlas-packet/teleportrock"
)

func TestErrorReasonToModeKey(t *testing.T) {
	cases := map[string]string{
		teleportrock2.ErrorReasonListFull:      trpkt.MapTransferModeMapNotAvailable,
		teleportrock2.ErrorReasonDuplicate:     trpkt.MapTransferModeMapNotAvailable,
		teleportrock2.ErrorReasonMapNotAllowed: trpkt.MapTransferModeMapNotAvailable,
		teleportrock2.ErrorReasonNotFound:      trpkt.MapTransferModeCannotGo,
	}
	for reason, want := range cases {
		if got := errorReasonToModeKey(reason); got != want {
			t.Errorf("%s: got %s want %s", reason, got, want)
		}
	}
}
