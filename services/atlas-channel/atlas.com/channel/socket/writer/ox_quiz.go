package writer

import (
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func OxQuizBody(enabled byte, category byte, number uint16) packet.Encode {
	return fieldcb.NewOxQuiz(enabled, category, number).Encode
}
