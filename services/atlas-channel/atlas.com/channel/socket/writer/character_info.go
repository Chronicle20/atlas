package writer

import (
	"atlas-channel/character"
	"atlas-channel/guild"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
)

const CharacterInfo = "CharacterInfo"

func CharacterInfoBody(tenant tenant.Model) func(c character.Model, g guild.Model) BodyProducer {
	return func(c character.Model, g guild.Model) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(c.Id())
			w.WriteByte(c.Level())
			w.WriteShort(c.JobId())
			w.WriteInt16(c.Fame())
			w.WriteBool(false) // marriage ring
			if g.Id() != 0 {
				w.WriteAsciiString(g.Name())
			} else {
				w.WriteAsciiString("")
			}
			w.WriteAsciiString("") // alliance name
			w.WriteByte(0)         // medal info
			w.WriteByte(0)         // pet activated, followed by pet info
			w.WriteByte(0)         // mount, followed by mount info
			w.WriteByte(0)         // wish list size
			if (tenant.Region() == "GMS" && tenant.MajorVersion() < 87) || tenant.Region() == "JMS" {
				w.WriteInt(0) // monster book level
				w.WriteInt(0) // normal card
				w.WriteInt(0) // special card
				w.WriteInt(0) // total cards
				w.WriteInt(0) // cover
			}
			if mi := c.Equipment().Medal().Equipable; mi != nil {
				w.WriteInt(mi.ItemId())
			} else {
				w.WriteInt(0)
			}
			w.WriteShort(0) // medal quests
			if (tenant.Region() == "GMS" && tenant.MajorVersion() > 83) || tenant.Region() == "JMS" {
				w.WriteInt(0) // chair
			}
			return w.Bytes()
		}
	}
}
