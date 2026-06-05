package handler

import (
	"atlas-channel/npc"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	npcclient "github.com/Chronicle20/atlas/libs/atlas-packet/npc/clientbound"
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

// bodyKindByMessageType is the protocol-invariant classification of which
// trailing body a serverbound continue-conversation packet carries, keyed by
// the *named* NPC conversation message type. The version-specific byte that
// names each type is supplied by tenant config ("messageType" table); only the
// name→body-kind grouping lives here, because it does not vary by version:
//
//   - ASK_TEXT / ASK_BOX_TEXT             → trailing text reply
//   - ASK_NUMBER / ASK_MENU / ASK_AVATAR /
//     ASK_SLIDE_MENU                      → trailing selection (number/index) reply
//   - SAY / ASK_YES_NO / ASK_YES_NO_QUEST
//     and anything unmapped               → no trailing body (answer is the action byte)
var bodyKindByMessageType = map[npcclient.NpcConversationMessageType]bodyKind{
	npcclient.NpcConversationMessageTypeAskText:      bodyText,
	npcclient.NpcConversationMessageTypeAskBoxText:   bodyText,
	npcclient.NpcConversationMessageTypeAskNumber:    bodySelection,
	npcclient.NpcConversationMessageTypeAskMenu:      bodySelection,
	npcclient.NpcConversationMessageTypeAskAvatar:    bodySelection,
	npcclient.NpcConversationMessageTypeAskSlideMenu: bodySelection,
}

// bodyKindFor resolves the wire lastMessageType byte to its body kind by
// reversing the tenant "messageType" table (byte→name) and classifying the
// name via bodyKindByMessageType. The byte numbering is never hardcoded here —
// it is whatever the tenant config assigns. Unknown or unconfigured bytes
// default to bodyNone so a misconfiguration cannot mis-parse the packet tail.
func bodyKindFor(l logrus.FieldLogger, readerOptions map[string]interface{}, msgType byte) bodyKind {
	name, ok := atlas_packet.ResolveName(l, readerOptions, "messageType", msgType)
	if !ok {
		l.Warnf("continue-conversation: lastMessageType [%d] is not present in the messageType config; treating as no trailing body. Verify the NPCContinueConversationHandle handler options carry the messageType table.", msgType)
		return bodyNone
	}
	return bodyKindByMessageType[npcclient.NpcConversationMessageType(name)]
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

		switch bodyKindFor(l, readerOptions, lastMessageType) {
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
