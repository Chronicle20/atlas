package writer

import (
	uipkt "github.com/Chronicle20/atlas-packet/ui"
	"github.com/Chronicle20/atlas-socket/packet"
)

const (
	UiDisable = "UiDisable"
)

func UiDisableBody(enable bool) packet.Encode {
	return uipkt.NewUiDisable(enable).Encode
}
