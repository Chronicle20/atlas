package writer

import (
	sw "github.com/Chronicle20/atlas-socket/writer"
	"github.com/sirupsen/logrus"
)

type BodyFunc = sw.BodyFunc

type Producer = sw.Producer

func getCode[E string](l logrus.FieldLogger) func(requester string, code E, codeProperty string, options map[string]interface{}) byte {
	return func(requester string, code E, codeProperty string, options map[string]interface{}) byte {
		var genericCodes interface{}
		var ok bool
		if genericCodes, ok = options[codeProperty]; !ok {
			l.Errorf("Code [%s] not configured for use in [%s]. Defaulting to 99 which will likely cause a client crash.", code, requester)
			return 99
		}

		var codes map[string]interface{}
		if codes, ok = genericCodes.(map[string]interface{}); !ok {
			l.Errorf("Code [%s] not configured for use in [%s]. Defaulting to 99 which will likely cause a client crash.", code, requester)
			return 99
		}

		res, ok := codes[string(code)].(float64)
		if !ok {
			l.Errorf("Code [%s] not configured for use in [%s]. Defaulting to 99 which will likely cause a client crash.", code, requester)
			return 99
		}
		return byte(res)
	}
}
