package writer

import (
	"github.com/Chronicle20/atlas-socket/packet"

	loginpkt "github.com/Chronicle20/atlas-packet/login"
)

const PicResult = "PicResult"

func PicResultBody() packet.Encode {
	return loginpkt.PicResult{}.Encode
}
