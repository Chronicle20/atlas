package clientbound

import (
	"context"

	"github.com/sirupsen/logrus"

	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	"github.com/Chronicle20/atlas/libs/atlas-packet/note"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

const (
	NoteOperationShow        = "SHOW"
	NoteOperationSendSuccess = "SEND_SUCCESS"
	NoteOperationSendError   = "SEND_ERROR"
	NoteOperationRefresh     = "REFRESH"

	NoteSendErrorReceiverOnline    = "RECEIVER_ONLINE"
	NoteSendErrorReceiverUnknown   = "RECEIVER_UNKNOWN"
	NoteSendErrorReceiverInboxFull = "RECEIVER_INBOX_FULL"

	// NoteSendErrorNoNoteItem is sent when a NOTE_ACTION SEND arrives from a
	// character owning no Note item (classification 509), or when the
	// note_send saga fails mid-flight. Tenant templates map it to code 3 —
	// outside the client's 0-2 dialog range: the mode-5 arm clears the
	// exclusive-request lock before decoding the code and shows no dialog
	// for unknown codes (IDA-verified, all nine versions). The SEND_ERROR
	// mode byte carrying this code is version-resolved (4 on v48/v61,
	// 5 on v72+) from the tenant operations table.
	NoteSendErrorNoNoteItem = "NO_NOTE_ITEM"
)

func NoteDisplayBody(entries []note.NoteEntry) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
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
			mode := atlas_packet.ResolveCode(l, options, "operations", NoteOperationSendError)
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
