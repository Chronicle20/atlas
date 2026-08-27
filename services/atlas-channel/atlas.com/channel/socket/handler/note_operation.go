package handler

import (
	"atlas-channel/character"
	"atlas-channel/compartment"
	"atlas-channel/note"
	"atlas-channel/session"
	model2 "atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	notepkt "github.com/Chronicle20/atlas/libs/atlas-packet/note"
	notecb "github.com/Chronicle20/atlas/libs/atlas-packet/note/clientbound"
	notesb "github.com/Chronicle20/atlas/libs/atlas-packet/note/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

const (
	NoteOperationSend    = "SEND"
	NoteOperationDiscard = "DISCARD"
	NoteOperationRequest = "REQUEST"
)

func NoteOperationHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := notesb.Operation{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		op := p.Op()
		np := note.NewProcessor(l, ctx)
		if isNoteOperation(l)(readerOptions, op, NoteOperationSend) {
			sp := &notesb.OperationSend{}
			sp.Decode(l, ctx)(r, readerOptions)

			// Gift-forward branch (giftFlag == 1): the only arm a legitimate
			// v83+ client ever writes (CCashShop::OnCashItemResLoadGiftDone).
			// It is paid for by the gift purchase, not a Note item, and must
			// not route through the consume-gated path below — see design
			// §2.2 and note_gift_forward.go.
			if sp.GiftFlag() == 1 {
				handleNoteGiftForward(l, ctx)(s, sp)
				return
			}

			// The tamper path (giftFlag == 0): no client writes it today.
			// Gate on Note-item ownership so a tampered client cannot mint
			// free notes (FR-4).
			cp, err := compartment.NewProcessor(l, ctx).GetByType(s.CharacterId(), inventory.TypeValueCash)
			if err != nil {
				l.WithError(err).Warnf("Character [%d] NOTE_ACTION SEND rejected: unable to load cash compartment.", s.CharacterId())
				_ = session.Announce(l)(ctx)(wp)(notecb.NoteOperationWriter)(notecb.NoteSendErrorBody(notecb.NoteSendErrorNoNoteItem))(s)
				return
			}
			a, found := cp.FindFirstByClassification(item.ClassificationNote)
			if !found {
				l.Warnf("Character [%d] attempted NOTE_ACTION SEND without owning a Note item (classification 509). Rejecting.", s.CharacterId())
				_ = session.Announce(l)(ctx)(wp)(notecb.NoteOperationWriter)(notecb.NoteSendErrorBody(notecb.NoteSendErrorNoNoteItem))(s)
				return
			}

			handleNoteSendRequest(l, ctx, wp)(s, a.TemplateId(), sp.ToName(), sp.Message())
			return
		}
		if isNoteOperation(l)(readerOptions, op, NoteOperationDiscard) {
			sp := &notesb.OperationDiscard{}
			sp.Decode(l, ctx)(r, readerOptions)
			l.Debugf("Character [%d] discarding [%d] notes. emptySlotCount [%d].", s.CharacterId(), sp.Count(), sp.EmptySlotCount())

			noteIds := make([]uint32, 0, sp.Count())

			for _, e := range sp.Entries() {
				l.Debugf("Character [%d] discarding note [%d]. flags [%d].", s.CharacterId(), e.Id(), e.Flag())

				// Validate the note exists and the flag matches, but never tear
				// down the session on a mismatch: a decode drift or a stale
				// client list must not crash the player to login (the actual
				// delete below is character-scoped, so a bogus id cannot remove
				// another character's note). Skip the offending entry and
				// discard whatever remains valid.
				n, err := np.GetById(e.Id())
				if err != nil {
					l.WithError(err).Warnf("Character [%d] cannot discard note [%d] (not found); skipping.", s.CharacterId(), e.Id())
					continue
				}

				if n.Flag() != e.Flag() {
					l.Warnf("Character [%d] discard of note [%d] has mismatched flag (expected [%d], got [%d]); skipping.", s.CharacterId(), e.Id(), n.Flag(), e.Flag())
					continue
				}

				noteIds = append(noteIds, e.Id())
			}

			if len(noteIds) == 0 {
				return
			}

			err := np.DiscardNotes(s.Field().Channel(), s.CharacterId(), noteIds)
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

			entries := make([]notepkt.NoteEntry, len(wnms))
			for i, n := range wnms {
				entries[i] = notepkt.NoteEntry{Id: n.Id, SenderName: n.SenderName, Message: n.Message, Timestamp: n.Timestamp, Flag: n.Flag}
			}
			err = session.Announce(l)(ctx)(wp)(notecb.NoteOperationWriter)(notecb.NoteDisplayBody(entries))(s)
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
