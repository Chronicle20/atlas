package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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
	// Legacy GMS (< v83) SPAWN_PLAYER wire divergences (task-113 stage E, v79
	// GMS_v79_1_DEVM.exe @13340): CUserRemote::Init sub_8D589E reads name as its
	// FIRST field (@0x8d58c9 DecodeStr) with NO leading Decode1 level byte, unlike
	// v83 CUserRemote::Init @0x97f55d (@0x97f589 Decode1 level) and v95 @0x955460
	// (m_nLevel = Decode1). v79 also has a single trailing effect byte (@0x8d5f67)
	// vs v83's two (@0x97fc33 + @0x97fd90), and no trailing team byte (base
	// CField::DecodeFieldSpecificData @0x513a15 forwards only the CUser, never the
	// packet). Leave v83/84/87/95/JMS unchanged.
	legacy := t.Region() == "GMS" && t.MajorVersion() < 83
	// Pre-v61 GMS (v48) SPAWN_PLAYER (CUserRemote::Init sub_6BBC17 @0x6bbc17,
	// GMS_v48_1_DEVM.exe) diverges further from the v79 legacy path: the CTS
	// foreign decode (sub_5CBA1F @0x6bbcde) goes STRAIGHT to AvatarLook::Decode
	// (@0x6bbcea) with NO Decode2(jobId) between; the pet section is a single
	// Decode1 flag + one pet (sub_58C7CC @0x6bbe5e), not the 3-slot bool loop;
	// and the ring tail is exactly six Decode1 flags (miniroom/adboard/couple/
	// friend/marriage/final-effect) with NO new-year-card byte. Leave v61+ (the
	// v79 legacy path) and all anchors unchanged.
	legacyV48 := t.Region() == "GMS" && t.MajorVersion() < 61
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		if !legacy {
			w.WriteByte(m.level)
		}
		w.WriteAsciiString(m.name)

		w.WriteAsciiString(m.guild.Name)
		w.WriteShort(m.guild.LogoBackground)
		w.WriteByte(m.guild.LogoBackgroundColor)
		w.WriteShort(m.guild.Logo)
		w.WriteByte(m.guild.LogoColor)

		w.WriteByteArray(m.cts.EncodeForeign(l, ctx)(options))
		if !legacyV48 {
			w.WriteShort(m.jobId)
		}
		w.WriteByteArray(m.avatar.Encode(l, ctx)(options))

		if (t.Region() == "GMS" && t.MajorVersion() > 87) || t.Region() == "JMS" {
			w.WriteInt(0) // driver id
			w.WriteInt(0) // passenger id
		}
		w.WriteInt(0) // choco count
		w.WriteInt(0) // item effect
		if t.IsRegion("GMS") && t.MajorAtLeast(87) {
			// v87+ nCompletedSetItemID; v84..86 == v83 (off-by-one fix). delta §3.1.7
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
		// bShowAdminEffect: GMS CUserRemote::Init reads a byte here before the pet
		// loop; the jms_v185 client (CUserRemote::Init @0xa52876) goes straight from
		// the foothold short into the pet while-loop with NO admin byte. IDA-verified
		// (jms export CUserRemote::Init call 18 foothold → call 19 pet terminator).
		if t.Region() != "JMS" {
			w.WriteByte(0) // bShowAdminEffect
		}

		if legacyV48 {
			// Pre-v61: single Decode1 flag then one pet (sub_58C7CC), no
			// terminator byte. The pet body (Int templateId, name, 8-byte SN,
			// x/y shorts, stance, foothold short, nameTag, chatBalloon) is
			// byte-identical to model.Pet.Encode.
			if len(m.pets) > 0 {
				w.WriteBool(true)
				w.WriteByteArray(m.pets[0].Pet.Encode(l, ctx)(options))
			} else {
				w.WriteByte(0)
			}
		} else {
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
		}

		w.WriteInt(1)  // mount level
		w.WriteInt(0)  // mount exp
		w.WriteInt(0)  // mount tiredness
		w.WriteByte(0) // mini room
		w.WriteByte(0) // ad board
		w.WriteByte(0) // couple ring
		w.WriteByte(0) // friendship ring
		w.WriteByte(0) // marriage ring

		if t.Region() == "GMS" && t.MajorVersion() >= 61 && t.MajorVersion() < 95 {
			// v48 (sub_6BBC17) has no new-year-card flag between marriage and the
			// final-effect byte — only 6 tail flags total.
			w.WriteByte(0) // new year card
		}

		w.WriteByte(0) // berserk / final-effect flag

		if t.Region() == "GMS" {
			if t.MajorVersion() >= 83 && t.MajorVersion() <= 87 {
				w.WriteByte(0) // v84..v87 2nd (dragon) effect byte; v79 Init has one effect byte only (@0x8d5f67)
			}
			if t.MajorVersion() > 87 {
				w.WriteByte(0) // new year card
				w.WriteInt(0)  // nPhase
			}
		} else if t.Region() == "JMS" {
			w.WriteByte(0) // final-effect flag (jms CUserRemote::Init call 47, last read)
		}
		// team (carnival) byte: GMS reads a trailing team byte; the jms_v185 client's
		// last packet read is the final-effect flag above (call 47) — no team byte.
		// v79 base CField::DecodeFieldSpecificData @0x513a15 forwards only the CUser
		// (not the packet) so legacy GMS reads no team byte either.
		if t.Region() != "JMS" && !legacy {
			w.WriteByte(0) // team
		}
		return w.Bytes()
	}
}

func (m CharacterSpawn) CharacterId() uint32              { return m.characterId }
func (m CharacterSpawn) Level() byte                      { return m.level }
func (m CharacterSpawn) Name() string                     { return m.name }
func (m CharacterSpawn) Guild() GuildEmblem               { return m.guild }
func (m CharacterSpawn) Cts() *model.CharacterTemporaryStat { return m.cts }
func (m CharacterSpawn) JobId() uint16                    { return m.jobId }
func (m CharacterSpawn) Avatar() model.Avatar             { return m.avatar }
func (m CharacterSpawn) Pets() []SpawnPet                 { return m.pets }
func (m CharacterSpawn) X() int16                         { return m.x }
func (m CharacterSpawn) Y() int16                         { return m.y }
func (m CharacterSpawn) Stance() byte                     { return m.stance }

func (m *CharacterSpawn) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		t := tenant.MustFromContext(ctx)
		legacy := t.Region() == "GMS" && t.MajorVersion() < 83
		legacyV48 := t.Region() == "GMS" && t.MajorVersion() < 61

		m.characterId = r.ReadUint32()
		if !legacy {
			m.level = r.ReadByte()
		}
		m.name = r.ReadAsciiString()

		m.guild.Name = r.ReadAsciiString()
		m.guild.LogoBackground = r.ReadUint16()
		m.guild.LogoBackgroundColor = r.ReadByte()
		m.guild.Logo = r.ReadUint16()
		m.guild.LogoColor = r.ReadByte()

		m.cts = model.NewCharacterTemporaryStat()
		m.cts.DecodeForeign(l, ctx)(r, options)

		if !legacyV48 {
			m.jobId = r.ReadUint16()
		}
		m.avatar.Decode(l, ctx)(r, options)

		if (t.Region() == "GMS" && t.MajorVersion() > 87) || t.Region() == "JMS" {
			_ = r.ReadUint32() // driver id
			_ = r.ReadUint32() // passenger id
		}
		_ = r.ReadUint32() // choco count
		_ = r.ReadUint32() // item effect
		if t.IsRegion("GMS") && t.MajorAtLeast(87) {
			// v87+ nCompletedSetItemID; v84..86 == v83 (off-by-one fix). delta §3.1.7
			_ = r.ReadUint32() // nCompletedSetItemID
		}
		_ = r.ReadUint32() // chair

		m.x = r.ReadInt16()
		m.y = r.ReadInt16()
		m.stance = r.ReadByte()

		_ = r.ReadUint16() // fh
		if t.Region() != "JMS" {
			_ = r.ReadByte() // bShowAdminEffect (GMS-only; jms has no admin byte)
		}

		m.pets = nil
		if legacyV48 {
			// Pre-v61: single Decode1 flag then at most one pet (sub_58C7CC).
			if r.ReadBool() {
				pet := model.Pet{}
				pet.Decode(l, ctx)(r, options)
				m.pets = append(m.pets, SpawnPet{Slot: 0, Pet: pet})
			}
		} else {
			// Pets: bool-terminated loop (true+data for each present pet, then false terminator)
			slot := int8(0)
			for r.ReadBool() {
				pet := model.Pet{}
				pet.Decode(l, ctx)(r, options)
				m.pets = append(m.pets, SpawnPet{Slot: slot, Pet: pet})
				slot++
			}
		}

		_ = r.ReadUint32() // mount level
		_ = r.ReadUint32() // mount exp
		_ = r.ReadUint32() // mount tiredness
		_ = r.ReadByte()   // mini room
		_ = r.ReadByte()   // ad board
		_ = r.ReadByte()   // couple ring
		_ = r.ReadByte()   // friendship ring
		_ = r.ReadByte()   // marriage ring

		if t.Region() == "GMS" && t.MajorVersion() >= 61 && t.MajorVersion() < 95 {
			_ = r.ReadByte() // new year card (absent pre-v61)
		}

		_ = r.ReadByte() // berserk / final-effect flag

		if t.Region() == "GMS" {
			if t.MajorVersion() >= 83 && t.MajorVersion() <= 87 {
				_ = r.ReadByte()
			}
			if t.MajorVersion() > 87 {
				_ = r.ReadByte()   // new year card
				_ = r.ReadUint32() // nPhase
			}
		} else if t.Region() == "JMS" {
			_ = r.ReadByte() // final-effect flag (jms last read)
		}
		if t.Region() != "JMS" && !legacy {
			_ = r.ReadByte() // team (GMS>=83-only)
		}
	}
}
