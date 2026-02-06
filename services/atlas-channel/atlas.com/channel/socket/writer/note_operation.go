package writer

import (
	"atlas-channel/socket/model"

	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const (
	NoteOperation            = "NoteOperation"
	NoteOperationShow        = "SHOW"         // 3
	NoteOperationSendSuccess = "SEND_SUCCESS" // 4
	NoteOperationSendError   = "SEND_ERROR"   // 5
	NoteOperationRefresh     = "REFRESH"

	NoteSendErrorReceiverOnline    = "RECEIVER_ONLINE"
	NoteSendErrorReceiverUnknown   = "RECEIVER_UNKNOWN"
	NoteSendErrorReceiverInboxFull = "RECEIVER_INBOX_FULL"
)

func NoteDisplayBody(l logrus.FieldLogger, t tenant.Model) func(notes []model.Note) BodyProducer {
	return func(notes []model.Note) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getNoteOperation(l)(options, NoteOperationShow))
			w.WriteByte(byte(len(notes)))
			for _, n := range notes {
				n.Encode(l, t, options)(w)
			}
			return w.Bytes()
		}
	}
}

func NoteSendSuccess(l logrus.FieldLogger) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteByte(getNoteOperation(l)(options, NoteOperationSendSuccess))
		return w.Bytes()
	}
}

func NoteSendError(l logrus.FieldLogger) func(error string) BodyProducer {
	return func(error string) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getNoteOperation(l)(options, NoteOperationSendSuccess))
			w.WriteByte(getNoteError(l)(options, error))
			return w.Bytes()
		}
	}
}

func NoteRefresh(l logrus.FieldLogger) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteByte(getNoteOperation(l)(options, NoteOperationRefresh))
		return w.Bytes()
	}
}

func getNoteOperation(l logrus.FieldLogger) func(options map[string]interface{}, key string) byte {
	return func(options map[string]interface{}, key string) byte {
		var genericCodes interface{}
		var ok bool
		if genericCodes, ok = options["operations"]; !ok {
			l.Errorf("Code [%s] not configured for use.", key)
			return 0
		}

		var codes map[string]interface{}
		if codes, ok = genericCodes.(map[string]interface{}); !ok {
			l.Errorf("Code [%s] not configured for use.", key)
			return 0
		}

		res, ok := codes[key].(float64)
		if !ok {
			l.Errorf("Code [%s] not configured for use.", key)
			return 0
		}
		return byte(res)
	}
}

func getNoteError(l logrus.FieldLogger) func(options map[string]interface{}, key string) byte {
	return func(options map[string]interface{}, key string) byte {
		var genericCodes interface{}
		var ok bool
		if genericCodes, ok = options["errors"]; !ok {
			l.Errorf("Code [%s] not configured for use.", key)
			return 0
		}

		var codes map[string]interface{}
		if codes, ok = genericCodes.(map[string]interface{}); !ok {
			l.Errorf("Code [%s] not configured for use.", key)
			return 0
		}

		res, ok := codes[key].(float64)
		if !ok {
			l.Errorf("Code [%s] not configured for use.", key)
			return 0
		}
		return byte(res)
	}
}
