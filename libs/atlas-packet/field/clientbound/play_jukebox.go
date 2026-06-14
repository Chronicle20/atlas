package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const PlayJukeboxWriter = "PlayJukebox"

// PlayJukebox is the clientbound CField::OnPlayJukeBox packet. The client reads a
// signed jukebox item id; only when that id is non-negative (itemId >= 0, the
// cash-item guard: a real item is playing rather than a stop signal) does it read
// the trailing player name string. A negative id stops the jukebox and carries no
// name. itemId is modelled as int32 to preserve that signed guard exactly.
type PlayJukebox struct {
	itemId     int32
	playerName string
}

func NewPlayJukebox(itemId int32, playerName string) PlayJukebox {
	return PlayJukebox{itemId: itemId, playerName: playerName}
}

func (m PlayJukebox) ItemId() int32      { return m.itemId }
func (m PlayJukebox) PlayerName() string { return m.playerName }

func (m PlayJukebox) Operation() string { return PlayJukeboxWriter }
func (m PlayJukebox) String() string {
	return fmt.Sprintf("itemId [%d] playerName [%s]", m.itemId, m.playerName)
}

func (m PlayJukebox) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(uint32(m.itemId))
		// The trailing player name is only present when the item id is non-negative.
		if m.itemId >= 0 {
			w.WriteAsciiString(m.playerName)
		}
		return w.Bytes()
	}
}

func (m *PlayJukebox) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.itemId = int32(r.ReadUint32())
		if m.itemId >= 0 {
			m.playerName = r.ReadAsciiString()
		}
	}
}
