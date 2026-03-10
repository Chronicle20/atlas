package writer

import (
	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/guild"
	"atlas-channel/pet"
	"atlas-channel/socket/model"
	"context"

	charpkt "github.com/Chronicle20/atlas-packet/character"
	packetmodel "github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterSpawn = "CharacterSpawn"

func CharacterSpawnBody(c character.Model, bs []buff.Model, g guild.Model, enteringField bool) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		t := tenant.MustFromContext(ctx)
		return func(options map[string]interface{}) []byte {
			ge := charpkt.GuildEmblem{}
			if g.Id() != 0 {
				ge.Name = g.Name()
				ge.LogoBackground = g.LogoBackground()
				ge.LogoBackgroundColor = g.LogoBackgroundColor()
				ge.Logo = g.Logo()
				ge.LogoColor = g.LogoColor()
			}

			cts := model.NewCharacterTemporaryStat()
			for _, b := range bs {
				for _, ch := range b.Changes() {
					cts.AddStat(l)(t)(ch.Type(), b.SourceId(), ch.Amount(), b.Level(), b.ExpiresAt())
				}
			}

			ava := model.NewFromCharacter(c, false)

			var pets []charpkt.SpawnPet
			if c.Pets() != nil {
				for _, p := range c.Pets() {
					pets = append(pets, charpkt.SpawnPet{
						Slot: p.Slot(),
						Pet: packetmodel.Pet{
							TemplateId:  p.TemplateId(),
							Name:        p.Name(),
							Id:          p.Id(),
							X:           p.X(),
							Y:           p.Y(),
							Stance:      p.Stance(),
							Foothold:    p.Fh(),
							NameTag:     0,
							ChatBalloon: 0,
						},
					})
				}
			}

			return charpkt.NewCharacterSpawn(
				c.Id(), c.Level(), c.Name(), ge, cts, uint16(c.JobId()), ava,
				pets, enteringField, c.X(), c.Y(), c.Stance(),
			).Encode(l, ctx)(options)
		}
	}
}

func writeForEachPet(w *response.Writer, ps []pet.Model, pe func(w *response.Writer, p pet.Model), pne func(w *response.Writer)) {
	for i := int8(0); i < 3; i++ {
		if ps == nil {
			pne(w)
			continue
		}

		var p *pet.Model
		for _, rp := range ps {
			if rp.Slot() == i {
				p = &rp
			}
		}
		if p != nil {
			pe(w, *p)
		} else {
			pne(w)
		}
	}
}

func writePetId(w *response.Writer, pet pet.Model) {
	w.WriteLong(uint64(pet.Id()))
}

func writeEmptyPetId(w *response.Writer) {
	w.WriteLong(0)
}
