package character

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterInfoWriter = "CharacterInfo"

// InfoPet represents a pet to be shown in the character info panel.
type InfoPet struct {
	Slot       int8
	TemplateId uint32
	Name       string
	Level      byte
	Closeness  uint16
	Fullness   byte
}

type CharacterInfo struct {
	characterId     uint32
	level           byte
	jobId           uint16
	fame            int16
	guildName       string
	pets            []InfoPet
	wishList        []uint32
	medalId         uint32
}

func NewCharacterInfo(characterId uint32, level byte, jobId uint16, fame int16, guildName string,
	pets []InfoPet, wishList []uint32, medalId uint32) CharacterInfo {
	return CharacterInfo{
		characterId: characterId, level: level, jobId: jobId, fame: fame,
		guildName: guildName, pets: pets, wishList: wishList, medalId: medalId,
	}
}

func (m CharacterInfo) Operation() string { return CharacterInfoWriter }
func (m CharacterInfo) String() string {
	return fmt.Sprintf("characterId [%d] level [%d] job [%d]", m.characterId, m.level, m.jobId)
}

func (m CharacterInfo) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		w.WriteByte(m.level)
		w.WriteShort(m.jobId)
		w.WriteInt16(m.fame)
		w.WriteBool(false) // marriage ring
		w.WriteAsciiString(m.guildName)
		w.WriteAsciiString("") // alliance name
		w.WriteByte(0)         // medal info

		// Pets: iterate 3 slots
		for slot := int8(0); slot < 3; slot++ {
			var found *InfoPet
			for i := range m.pets {
				if m.pets[i].Slot == slot {
					found = &m.pets[i]
					break
				}
			}
			if found != nil {
				w.WriteBool(true)
				w.WriteInt(found.TemplateId)
				w.WriteAsciiString(found.Name)
				w.WriteByte(found.Level)
				w.WriteShort(found.Closeness)
				w.WriteByte(found.Fullness)
				w.WriteShort(0) // skill
				w.WriteInt(0)   // itemId
			}
		}
		w.WriteBool(false) // more pets?

		w.WriteByte(0) // mount

		w.WriteByte(byte(len(m.wishList)))
		for _, sn := range m.wishList {
			w.WriteInt(sn)
		}

		if (t.Region() == "GMS" && t.MajorVersion() < 87) || t.Region() == "JMS" {
			w.WriteInt(0) // monster book level
			w.WriteInt(0) // normal card
			w.WriteInt(0) // special card
			w.WriteInt(0) // total cards
			w.WriteInt(0) // cover
		}

		w.WriteInt(m.medalId)
		w.WriteShort(0) // medal quests
		if (t.Region() == "GMS" && t.MajorVersion() > 83) || t.Region() == "JMS" {
			w.WriteInt(0) // chair
		}
		return w.Bytes()
	}
}

func (m *CharacterInfo) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		// No-op: CharacterInfo is server-send-only with complex conditional encoding.
	}
}
