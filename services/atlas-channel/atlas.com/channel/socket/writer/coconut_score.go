package writer

import (
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

func CoconutScoreBody(mapleScore uint16, storyScore uint16) packet.Encode {
	return fieldcb.NewCoconutScore(mapleScore, storyScore).Encode
}
