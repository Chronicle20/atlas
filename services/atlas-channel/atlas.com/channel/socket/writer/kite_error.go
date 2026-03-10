package writer

import (
	fieldpkt "github.com/Chronicle20/atlas-packet/field"
	"github.com/Chronicle20/atlas-socket/packet"
)

const (
	SpawnKiteError = "SpawnKiteError"
)

func SpawnKiteErrorBody() packet.Encode {
	return fieldpkt.NewKiteError().Encode
}
