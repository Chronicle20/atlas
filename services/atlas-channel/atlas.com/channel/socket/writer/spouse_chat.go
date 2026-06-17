package writer

import (
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func SpouseChatBody(mode byte, sender string, flag byte, chatText string, partnerFlag byte, partnerText string) packet.Encode {
	return fieldcb.NewSpouseChat(mode, sender, flag, chatText, partnerFlag, partnerText).Encode
}
