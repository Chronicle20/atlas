package writer

import (
	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/guild"
	"atlas-channel/pet"
	"atlas-channel/socket/model"
	"context"

	packetmodel "github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterSpawn = "CharacterSpawn"

func CharacterSpawnBody(c character.Model, bs []buff.Model, g guild.Model, enteringField bool) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		t := tenant.MustFromContext(ctx)
		return func(options map[string]interface{}) []byte {
			w.WriteInt(c.Id())
			w.WriteByte(c.Level())
			w.WriteAsciiString(c.Name())
			if g.Id() != 0 {
				w.WriteAsciiString(g.Name())
				w.WriteShort(g.LogoBackground())
				w.WriteByte(g.LogoBackgroundColor())
				w.WriteShort(g.Logo())
				w.WriteByte(g.LogoColor())
			} else {
				w.WriteAsciiString("")
				w.WriteShort(0)
				w.WriteByte(0)
				w.WriteShort(0)
				w.WriteByte(0)
			}

			cts := model.NewCharacterTemporaryStat()
			for _, b := range bs {
				for _, ch := range b.Changes() {
					cts.AddStat(l)(t)(ch.Type(), b.SourceId(), ch.Amount(), b.Level(), b.ExpiresAt())
				}
			}
			w.WriteByteArray(cts.EncodeForeign(l, ctx)(options))
			w.WriteShort(uint16(c.JobId()))

			ava := model.NewFromCharacter(c, false)
			w.WriteByteArray(ava.Encode(l, ctx)(options))

			if (t.Region() == "GMS" && t.MajorVersion() > 87) || t.Region() == "JMS" {
				w.WriteInt(0) // driver id
				w.WriteInt(0) // passenger id
			}
			w.WriteInt(0) // choco count
			w.WriteInt(0) // item effect
			if t.Region() == "GMS" && t.MajorVersion() > 83 {
				w.WriteInt(0) // nCompletedSetItemID
			}
			w.WriteInt(0) // chair

			if enteringField {
				w.WriteInt16(c.X())
				w.WriteInt16(c.Y() - 42)
				w.WriteByte(6) // move action / stance
			} else {
				w.WriteInt16(c.X())
				w.WriteInt16(c.Y())
				w.WriteByte(c.Stance()) // move action / stance
			}

			w.WriteShort(0) // fh
			w.WriteByte(0)  // bShowAdminEffect

			// TODO clean this up.
			writeForEachPet(w, c.Pets(), func(w *response.Writer, p pet.Model) {
				m := packetmodel.Pet{
					TemplateId:  p.TemplateId(),
					Name:        p.Name(),
					Id:          p.Id(),
					X:           p.X(),
					Y:           p.Y(),
					Stance:      p.Stance(),
					Foothold:    p.Fh(),
					NameTag:     0,
					ChatBalloon: 0,
				}
				w.WriteBool(true)
				w.WriteByteArray(m.Encode(l, ctx)(options))
			}, func(w *response.Writer) {
			})
			w.WriteByte(0) // end of pets

			w.WriteInt(1)  // mount level
			w.WriteInt(0)  // mount exp
			w.WriteInt(0)  // mount tiredness
			w.WriteByte(0) // mini room
			w.WriteByte(0) // ad board

			// TODO GMS - JMS have different ring encoding/decoding
			w.WriteByte(0) // couple ring
			w.WriteByte(0) // friendship ring
			w.WriteByte(0) // marriage ring

			if t.Region() == "GMS" && t.MajorVersion() < 95 {
				w.WriteByte(0) // new year card
			}

			w.WriteByte(0) // berserk

			if t.Region() == "GMS" {
				if t.MajorVersion() <= 87 {
					w.WriteByte(0) // unknown (same as JMS unknown)
				}
				if t.MajorVersion() > 87 {
					w.WriteByte(0) // new year card
					w.WriteInt(0)  // nPhase
				}
			} else if t.Region() == "JMS" {
				w.WriteByte(0) // unknown
			}
			w.WriteByte(0) // team
			return w.Bytes()
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
