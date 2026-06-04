package handler

import (
	"atlas-channel/npc"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	npc2 "github.com/Chronicle20/atlas/libs/atlas-packet/npc/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

type bodyKind int

const (
	bodyNone bodyKind = iota
	bodyText
	bodySelection
)

// bodyKindFor maps the client's lastMessageType to the trailing body the
// serverbound continue-conversation packet carries (task-080 B2.1).
//
//	3 (OnAskText) / 14 (OnAskBoxText) → text reply
//	5 (OnAskMenu) / 8 (OnAskAvatar) / 9 → selection
//	0/1/2/13 (Say/AskYesNo) → no trailing body
func bodyKindFor(msgType byte) bodyKind {
	switch msgType {
	case 3, 14:
		return bodyText
	case 5, 8, 9:
		return bodySelection
	default:
		return bodyNone
	}
}

func NPCContinueConversationHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := npc2.ContinueConversation{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		lastMessageType := p.LastMessageType()
		action := p.Action()
		//returnText := ""
		selection := int32(-1)

		switch bodyKindFor(lastMessageType) {
		case bodyText:
			if action != 0 {
				sp := &npc2.ContinueConversationText{}
				sp.Decode(l, ctx)(r, readerOptions)
				// TODO handle quest in progress, continue quest

				//TODO set return text
				_ = npc.NewProcessor(l, ctx).ContinueConversation(s.CharacterId(), action, lastMessageType, selection)
				return
			}
			// TODO handle quest in progress, dispose
			_ = npc.NewProcessor(l, ctx).DisposeConversation(s.CharacterId())
			return
		case bodySelection:
			sp := &npc2.ContinueConversationSelection{}
			sp.Decode(l, ctx)(r, readerOptions)
			selection = sp.Selection()
			// TODO handle quest in progress, continue quest
			_ = npc.NewProcessor(l, ctx).ContinueConversation(s.CharacterId(), action, lastMessageType, selection)
		default: // bodyNone: Say/AskYesNo carry no trailing body
			if action != 0 {
				// TODO handle quest in progress, continue quest
				_ = npc.NewProcessor(l, ctx).ContinueConversation(s.CharacterId(), action, lastMessageType, selection)
				return
			}
			// TODO handle quest in progress, dispose
			_ = npc.NewProcessor(l, ctx).DisposeConversation(s.CharacterId())
		}
	}
}
