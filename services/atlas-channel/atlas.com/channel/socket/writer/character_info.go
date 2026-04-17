package writer

import (
	"atlas-channel/cashshop/wishlist"
	"atlas-channel/character"
	"atlas-channel/guild"
	"context"

	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)


func CharacterInfoBody(c character.Model, g guild.Model, wl []wishlist.Model) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			guildName := ""
			if g.Id() != 0 {
				guildName = g.Name()
			}

			var pets []charpkt.InfoPet
			if c.Pets() != nil {
				for _, p := range c.Pets() {
					pets = append(pets, charpkt.InfoPet{
						Slot:       p.Slot(),
						TemplateId: p.TemplateId(),
						Name:       p.Name(),
						Level:      p.Level(),
						Closeness:  p.Closeness(),
						Fullness:   p.Fullness(),
					})
				}
			}

			var wishListSNs []uint32
			for _, i := range wl {
				wishListSNs = append(wishListSNs, i.SerialNumber())
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

			return charpkt.NewCharacterInfo(
				c.Id(), c.Level(), uint16(c.JobId()), c.Fame(), guildName,
				pets, wishListSNs, medalId,
			).Encode(l, ctx)(options)
		}
	}
}

