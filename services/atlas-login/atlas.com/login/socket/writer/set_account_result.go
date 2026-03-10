package writer

import (
	"github.com/Chronicle20/atlas-socket/packet"

	loginpkt "github.com/Chronicle20/atlas-packet/login"
)

const SetAccountResult = "SetAccountResult"

func SetAccountResultBody(gender byte, success bool) packet.Encode {
	return loginpkt.NewSetAccountResult(gender, success).Encode
}
