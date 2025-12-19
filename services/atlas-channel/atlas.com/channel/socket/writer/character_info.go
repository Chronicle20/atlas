package writer

import (
	"atlas-channel/cashshop/wishlist"
	"atlas-channel/character"
	"atlas-channel/guild"
	"atlas-channel/pet"
	"github.com/Chronicle20/atlas-constants/inventory/slot"

	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
)

const CharacterInfo = "CharacterInfo"

func CharacterInfoBody(tenant tenant.Model) func(c character.Model, g guild.Model, wl []wishlist.Model) BodyProducer {
	return func(c character.Model, g guild.Model, wl []wishlist.Model) BodyProducer {
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

			// TODO pet skill and item writing
			writeForEachPet(w, c.Pets(), func(w *response.Writer, p pet.Model) {
				w.WriteBool(true)
				w.WriteInt(p.TemplateId())
				w.WriteAsciiString(p.Name())
				w.WriteByte(p.Level())
				w.WriteShort(p.Closeness())
				w.WriteByte(p.Fullness())
				w.WriteShort(0) // skill
				w.WriteInt(0)   // itemId
			}, func(w *response.Writer) {
			})
			w.WriteBool(false) // more pets?

			w.WriteByte(0) // mount, followed by mount info

			w.WriteByte(byte(len(wl)))
			for _, i := range wl {
				w.WriteInt(i.SerialNumber())
			}

			if (tenant.Region() == "GMS" && tenant.MajorVersion() < 87) || tenant.Region() == "JMS" {
				w.WriteInt(0) // monster book level
				w.WriteInt(0) // normal card
				w.WriteInt(0) // special card
				w.WriteInt(0) // total cards
				w.WriteInt(0) // cover
			}

			medalId := uint32(0)
			ms, err := slot.GetSlotByType("medal")
			if err == nil {
				if em, ok := c.Equipment().Get(ms.Type); ok {
					if me := em.Equipable; me != nil {
						medalId = me.TemplateId()
					}
				}
			}
			w.WriteInt(medalId)

			w.WriteShort(0) // medal quests
			if (tenant.Region() == "GMS" && tenant.MajorVersion() > 83) || tenant.Region() == "JMS" {
				w.WriteInt(0) // chair
			}
			return w.Bytes()
		}
	}
}
