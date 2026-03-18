package writer

import (
	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/guild"
	"atlas-channel/socket/model"
	"context"

	charpkt "github.com/Chronicle20/atlas-packet/character/clientbound"
	packetmodel "github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)


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

			cts := packetmodel.NewCharacterTemporaryStat()
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
