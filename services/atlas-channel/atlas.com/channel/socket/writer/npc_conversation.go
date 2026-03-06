package writer

import (
	"atlas-channel/socket/model"

	"github.com/Chronicle20/atlas-socket/packet"
)

const NPCConversation = "NPCConversation"

func NPCConversationBody(c model.NpcConversation) packet.Encode {
	return c.Encoder
}
