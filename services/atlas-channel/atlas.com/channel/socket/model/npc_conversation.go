package model

import (
	"context"

	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	npcpkt "github.com/Chronicle20/atlas/libs/atlas-packet/npc/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

type NpcConversation struct {
	SpeakerTypeId          byte
	SpeakerTemplateId      uint32
	SecondaryNpcTemplateId uint32
	MsgType                npcpkt.NpcConversationMessageType
	Param                  byte
	ConversationDetail     packet.Encoder
}

func NewNpcConversation(npcId uint32, msgType npcpkt.NpcConversationMessageType, speakerByte byte, secondaryNpcId uint32, conversationDetail packet.Encoder) NpcConversation {
	return NpcConversation{
		SpeakerTypeId:          speakerByte,
		SpeakerTemplateId:      npcId,
		SecondaryNpcTemplateId: secondaryNpcId,
		MsgType:                msgType,
		Param:                  speakerByte,
		ConversationDetail:     conversationDetail,
	}
}

func (b *NpcConversation) Encoder(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	return func(options map[string]interface{}) []byte {
		msgTypeByte := atlas_packet.ResolveCode(l, options, "messageType", string(b.MsgType))
		detailBytes := b.ConversationDetail.Encode(l, ctx)(options)
		pkt := npcpkt.NewNpcConversation(b.SpeakerTypeId, b.SpeakerTemplateId, msgTypeByte, b.Param, b.SecondaryNpcTemplateId, detailBytes)
		return pkt.Encode(l, ctx)(options)
	}
}
