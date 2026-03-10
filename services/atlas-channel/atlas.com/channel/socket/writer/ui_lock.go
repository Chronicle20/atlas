package writer

import (
	uipkt "github.com/Chronicle20/atlas-packet/ui"
	"github.com/Chronicle20/atlas-socket/packet"
)

const (
	UiLock = "UiLock"
)

func UiLockBody(enable bool, tAfterLeaveDirectionMode int32) packet.Encode {
	return uipkt.NewUiLock(enable, tAfterLeaveDirectionMode).Encode
}
