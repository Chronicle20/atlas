package character

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterSpawnWriter = "CharacterSpawn"

// SpawnPet represents a pet to be shown with a spawning character.
type SpawnPet struct {
	Slot int8
	Pet  model.Pet
}

// GuildEmblem holds guild display info for a character spawn.
type GuildEmblem struct {
	Name                string
	LogoBackground      uint16
	LogoBackgroundColor byte
	Logo                uint16
	LogoColor           byte
}

type CharacterSpawn struct {
	characterId   uint32
	level         byte
	name          string
	guild         GuildEmblem
	cts           *model.CharacterTemporaryStat
	jobId         uint16
	avatar        model.Avatar
	pets          []SpawnPet
	enteringField bool
	x             int16
	y             int16
	stance        byte
}

func NewCharacterSpawn(characterId uint32, level byte, name string, guild GuildEmblem,
	cts *model.CharacterTemporaryStat, jobId uint16, avatar model.Avatar,
	pets []SpawnPet, enteringField bool, x int16, y int16, stance byte) CharacterSpawn {
	return CharacterSpawn{
		characterId: characterId, level: level, name: name, guild: guild,
		cts: cts, jobId: jobId, avatar: avatar, pets: pets,
		enteringField: enteringField, x: x, y: y, stance: stance,
	}
}

func (m CharacterSpawn) Operation() string { return CharacterSpawnWriter }
func (m CharacterSpawn) String() string {
	return fmt.Sprintf("characterId [%d] name [%s]", m.characterId, m.name)
}

func (m CharacterSpawn) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		w.WriteByte(m.level)
		w.WriteAsciiString(m.name)

		w.WriteAsciiString(m.guild.Name)
		w.WriteShort(m.guild.LogoBackground)
		w.WriteByte(m.guild.LogoBackgroundColor)
		w.WriteShort(m.guild.Logo)
		w.WriteByte(m.guild.LogoColor)

		w.WriteByteArray(m.cts.EncodeForeign(l, ctx)(options))
		w.WriteShort(m.jobId)
		w.WriteByteArray(m.avatar.Encode(l, ctx)(options))

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

		if m.enteringField {
			w.WriteInt16(m.x)
			w.WriteInt16(m.y - 42)
			w.WriteByte(6)
		} else {
			w.WriteInt16(m.x)
			w.WriteInt16(m.y)
			w.WriteByte(m.stance)
		}

		w.WriteShort(0) // fh
		w.WriteByte(0)  // bShowAdminEffect

		// Pets: iterate 3 slots
		for slot := int8(0); slot < 3; slot++ {
			var found *SpawnPet
			for i := range m.pets {
				if m.pets[i].Slot == slot {
					found = &m.pets[i]
					break
				}
			}
			if found != nil {
				w.WriteBool(true)
				w.WriteByteArray(found.Pet.Encode(l, ctx)(options))
			}
		}
		w.WriteByte(0) // end of pets

		w.WriteInt(1)  // mount level
		w.WriteInt(0)  // mount exp
		w.WriteInt(0)  // mount tiredness
		w.WriteByte(0) // mini room
		w.WriteByte(0) // ad board
		w.WriteByte(0) // couple ring
		w.WriteByte(0) // friendship ring
		w.WriteByte(0) // marriage ring

		if t.Region() == "GMS" && t.MajorVersion() < 95 {
			w.WriteByte(0) // new year card
		}

		w.WriteByte(0) // berserk

		if t.Region() == "GMS" {
			if t.MajorVersion() <= 87 {
				w.WriteByte(0)
			}
			if t.MajorVersion() > 87 {
				w.WriteByte(0) // new year card
				w.WriteInt(0)  // nPhase
			}
		} else if t.Region() == "JMS" {
			w.WriteByte(0)
		}
		w.WriteByte(0) // team
		return w.Bytes()
	}
}

func (m *CharacterSpawn) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		// No-op: CharacterSpawn is server-send-only with complex conditional encoding.
	}
}
