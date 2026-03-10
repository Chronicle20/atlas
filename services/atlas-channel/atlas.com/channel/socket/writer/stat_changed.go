package writer

import (
	"atlas-channel/socket/model"

	statpkt "github.com/Chronicle20/atlas-packet/stat"
	"github.com/Chronicle20/atlas-socket/packet"
)

const StatChanged = "StatChanged"

func StatChangedBody(updates []model.StatUpdate, exclRequestSent bool) packet.Encode {
	pktUpdates := make([]statpkt.Update, len(updates))
	for i, u := range updates {
		pktUpdates[i] = statpkt.NewUpdate(u.Stat(), u.Value())
	}
	return statpkt.NewStatChanged(pktUpdates, exclRequestSent).Encode
}
