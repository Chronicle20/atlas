package writer

import (
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func SetQuestTimeBody(quests []fieldcb.QuestTime) packet.Encode {
	return fieldcb.NewSetQuestTime(quests).Encode
}
