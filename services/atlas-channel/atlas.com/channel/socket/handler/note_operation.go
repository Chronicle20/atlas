package handler

import (
	"atlas-channel/character"
	"atlas-channel/note"
	"atlas-channel/session"
	model2 "atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const (
	NoteOperationHandle = "NoteOperationHandle"

	NoteOperationSend    = "SEND"
	NoteOperationDiscard = "DISCARD"
	NoteOperationRequest = "REQUEST"
)

func NoteOperationHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		op := r.ReadByte()
		np := note.NewProcessor(l, ctx)
		if isNoteOperation(l)(readerOptions, op, NoteOperationSend) {
			toName := r.ReadAsciiString()
			message := r.ReadAsciiString()

			tc, err := character.NewProcessor(l, ctx).GetByName(toName)
			if err != nil {
				l.WithError(err).Errorf("Unable to locate character by name [%s] to send note to.", toName)
				_ = session.Announce(l)(ctx)(wp)(writer.NoteOperation)(writer.NoteSendError(l)(writer.NoteSendErrorReceiverUnknown))(s)
				return
			}

			err = np.SendNote(s.CharacterId(), tc.Id(), message, 1)
			if err != nil {
				l.WithError(err).Errorf("Character [%d] unable to send note.", s.CharacterId())
			}
			return
		}
		if isNoteOperation(l)(readerOptions, op, NoteOperationDiscard) {
			count := r.ReadByte()
			val1 := r.ReadByte()
			val2 := r.ReadByte()
			l.Debugf("Character [%d] discarding [%d] notes. val1 [%d], val2 [%d].", s.CharacterId(), count, val1, val2)

			noteIds := make([]uint32, 0, count)

			for i := byte(0); i < count; i++ {
				id := r.ReadUint32()
				flag := r.ReadByte()
				l.Debugf("Character [%d] discarding note [%d]. flags [%d].", s.CharacterId(), id, flag)

				// Verify the note exists and the flag matches
				n, err := np.GetById(id)
				if err != nil {
					l.WithError(err).Errorf("Character [%d] unable to retrieve note [%d].", s.CharacterId(), id)
					_ = session.NewProcessor(l, ctx).Destroy(s)
					return
				}

				if n.Flag() != flag {
					l.Errorf("Character [%d] attempting to discard note [%d] with incorrect flag. Expected [%d], got [%d].", s.CharacterId(), id, n.Flag(), flag)
					_ = session.NewProcessor(l, ctx).Destroy(s)
					return
				}

				noteIds = append(noteIds, id)
			}

			err := np.DiscardNotes(s.CharacterId(), noteIds)
			if err != nil {
				l.WithError(err).Errorf("Character [%d] unable to discard notes.", s.CharacterId())
			}
			return
		}
		if isNoteOperation(l)(readerOptions, op, NoteOperationRequest) {
			var nms []note.Model
			nms, err := note.NewProcessor(l, ctx).GetByCharacter(s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Unable to read notes for character [%d].", s.CharacterId())
				return
			}
			if len(nms) == 0 {
				return
			}

			cnm := make(map[uint32]string)

			var wnms []model2.Note
			wnms, err = model.SliceMap(func(m note.Model) (model2.Note, error) {
				var sn string
				var ok bool
				if sn, ok = cnm[m.SenderId()]; !ok {
					var c character.Model
					c, err = character.NewProcessor(l, ctx).GetById()(m.SenderId())
					if err != nil {
						cnm[m.SenderId()] = "Unknown"
						sn = "Unknown"
					} else {
						cnm[m.SenderId()] = c.Name()
						sn = c.Name()
					}
				}

				return model2.Note{
					Id:         m.Id(),
					SenderName: sn,
					Message:    m.Message(),
					Timestamp:  m.Timestamp(),
					Flag:       m.Flag(),
				}, nil
			})(model.FixedProvider(nms))(model.ParallelMap())()

			err = session.Announce(l)(ctx)(wp)(writer.NoteOperation)(writer.NoteDisplayBody(l, tenant.MustFromContext(ctx))(wnms))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to show key map for character [%d].", s.CharacterId())
			}
		}

		l.Debugf("Character [%d] attempting to perform note operation [%d].", s.CharacterId(), op)
	}
}

func isNoteOperation(l logrus.FieldLogger) func(options map[string]interface{}, op byte, key string) bool {
	return func(options map[string]interface{}, op byte, key string) bool {
		var genericCodes interface{}
		var ok bool
		if genericCodes, ok = options["operations"]; !ok {
			l.Errorf("Code [%s] not configured for use.", key)
			return false
		}

		var codes map[string]interface{}
		if codes, ok = genericCodes.(map[string]interface{}); !ok {
			l.Errorf("Code [%s] not configured for use.", key)
			return false
		}

		res, ok := codes[key].(float64)
		if !ok {
			l.Errorf("Code [%s] not configured for use.", key)
			return false
		}
		return byte(res) == op
	}
}
