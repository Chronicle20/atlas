package writer

import (
	atlas_packet "github.com/Chronicle20/atlas-packet"
	sw "github.com/Chronicle20/atlas-socket/writer"
	"github.com/sirupsen/logrus"
)

type BodyFunc = sw.BodyFunc

type Producer = sw.Producer

func getCode[E string](l logrus.FieldLogger) func(requester string, code E, codeProperty string, options map[string]interface{}) byte {
	return func(requester string, code E, codeProperty string, options map[string]interface{}) byte {
		return atlas_packet.ResolveCode(l, options, codeProperty, string(code))
	}
}
