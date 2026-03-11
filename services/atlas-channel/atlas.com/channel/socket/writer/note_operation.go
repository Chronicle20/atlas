package writer

import (
	"atlas-channel/socket/model"
	"context"

	atlas_packet "github.com/Chronicle20/atlas-packet"
	notepkt "github.com/Chronicle20/atlas-packet/note"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

const (
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
			entries := make([]notepkt.NoteEntry, len(notes))
			for i, n := range notes {
				entries[i] = notepkt.NoteEntry{
					Id:         n.Id,
					SenderName: n.SenderName,
					Message:    n.Message,
					Timestamp:  n.Timestamp,
					Flag:       n.Flag,
				}
			}
			return notepkt.NewNoteDisplay(mode, entries).Encode(l, ctx)(options)
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
		return atlas_packet.ResolveCode(l, options, "operations", key)
	}
}

func getNoteError(l logrus.FieldLogger) func(options map[string]interface{}, key string) byte {
	return func(options map[string]interface{}, key string) byte {
		return atlas_packet.ResolveCode(l, options, "errors", key)
	}
}
