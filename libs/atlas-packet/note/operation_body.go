package note

import (
	"context"

	atlas_packet "github.com/Chronicle20/atlas-packet"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

const (
	NoteOperationShow        = "SHOW"
	NoteOperationSendSuccess = "SEND_SUCCESS"
	NoteOperationSendError   = "SEND_ERROR"
	NoteOperationRefresh     = "REFRESH"

	NoteSendErrorReceiverOnline    = "RECEIVER_ONLINE"
	NoteSendErrorReceiverUnknown   = "RECEIVER_UNKNOWN"
	NoteSendErrorReceiverInboxFull = "RECEIVER_INBOX_FULL"
)

func NoteDisplayBody(entries []NoteEntry) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", NoteOperationShow, func(mode byte) packet.Encoder {
		return NewNoteDisplay(mode, entries)
	})
}

func NoteSendSuccessBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", NoteOperationSendSuccess, func(mode byte) packet.Encoder {
		return NewNoteSendSuccess(mode)
	})
}

func NoteSendErrorBody(errorKey string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := atlas_packet.ResolveCode(l, options, "operations", NoteOperationSendSuccess)
			errorCode := atlas_packet.ResolveCode(l, options, "errors", errorKey)
			return NewNoteSendError(mode, errorCode).Encode(l, ctx)(options)
		}
	}
}

func NoteRefreshBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", NoteOperationRefresh, func(mode byte) packet.Encoder {
		return NewNoteRefresh(mode)
	})
}
