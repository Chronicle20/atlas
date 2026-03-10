package writer

import (
	"atlas-channel/socket/model"
	"context"

	notepkt "github.com/Chronicle20/atlas-packet/note"
	"github.com/Chronicle20/atlas-socket/packet"
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

func NoteDisplayBody(notes []model.Note) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getNoteOperation(l)(options, NoteOperationShow)
			noteBytes := make([][]byte, len(notes))
			for i, n := range notes {
				noteBytes[i] = n.Encoder(l, ctx)(options)
			}
			return notepkt.NewNoteDisplay(mode, noteBytes).Encode(l, ctx)(options)
		}
	}
}

func NoteSendSuccess() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getNoteOperation(l)(options, NoteOperationSendSuccess)
			return notepkt.NewNoteSendSuccess(mode).Encode(l, ctx)(options)
		}
	}
}

func NoteSendError(error string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getNoteOperation(l)(options, NoteOperationSendSuccess)
			errorCode := getNoteError(l)(options, error)
			return notepkt.NewNoteSendError(mode, errorCode).Encode(l, ctx)(options)
		}
	}
}

func NoteRefresh() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getNoteOperation(l)(options, NoteOperationRefresh)
			return notepkt.NewNoteRefresh(mode).Encode(l, ctx)(options)
		}
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
