package model

import (
	"context"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type CharacterStatistics struct {
	id                 uint32
	name               string
	gender             byte
	skinColor          byte
	face               uint32
	hair               uint32
	petIds             [3]uint64
	level              byte
	jobId              uint16
	strength           uint16
	dexterity          uint16
	intelligence       uint16
	luck               uint16
	hp                 uint16
	maxHp              uint16
	mp                 uint16
	maxMp              uint16
	ap                 uint16
	hasSPTable         bool
	sp                 uint16
	experience         uint32
	fame               int16
	gachaponExperience uint32
	mapId              uint32
	spawnPoint         byte
}

func NewCharacterStatistics(
	id uint32, name string, gender byte, skinColor byte, face uint32, hair uint32,
	petIds [3]uint64, level byte, jobId uint16,
	strength uint16, dexterity uint16, intelligence uint16, luck uint16,
	hp uint16, maxHp uint16, mp uint16, maxMp uint16,
	ap uint16, hasSPTable bool, sp uint16,
	experience uint32, fame int16, gachaponExperience uint32,
	mapId uint32, spawnPoint byte,
) CharacterStatistics {
	return CharacterStatistics{
		id: id, name: name, gender: gender, skinColor: skinColor,
		face: face, hair: hair, petIds: petIds,
		level: level, jobId: jobId,
		strength: strength, dexterity: dexterity, intelligence: intelligence, luck: luck,
		hp: hp, maxHp: maxHp, mp: mp, maxMp: maxMp,
		ap: ap, hasSPTable: hasSPTable, sp: sp,
		experience: experience, fame: fame, gachaponExperience: gachaponExperience,
		mapId: mapId, spawnPoint: spawnPoint,
	}
}

func (m CharacterStatistics) Id() uint32                 { return m.id }
func (m CharacterStatistics) Name() string               { return m.name }
func (m CharacterStatistics) Gender() byte               { return m.gender }
func (m CharacterStatistics) SkinColor() byte            { return m.skinColor }
func (m CharacterStatistics) Face() uint32               { return m.face }
func (m CharacterStatistics) Hair() uint32               { return m.hair }
func (m CharacterStatistics) PetIds() [3]uint64          { return m.petIds }
func (m CharacterStatistics) Level() byte                { return m.level }
func (m CharacterStatistics) JobId() uint16              { return m.jobId }
func (m CharacterStatistics) Strength() uint16           { return m.strength }
func (m CharacterStatistics) Dexterity() uint16          { return m.dexterity }
func (m CharacterStatistics) Intelligence() uint16       { return m.intelligence }
func (m CharacterStatistics) Luck() uint16               { return m.luck }
func (m CharacterStatistics) Hp() uint16                 { return m.hp }
func (m CharacterStatistics) MaxHp() uint16              { return m.maxHp }
func (m CharacterStatistics) Mp() uint16                 { return m.mp }
func (m CharacterStatistics) MaxMp() uint16              { return m.maxMp }
func (m CharacterStatistics) Ap() uint16                 { return m.ap }
func (m CharacterStatistics) HasSPTable() bool           { return m.hasSPTable }
func (m CharacterStatistics) Sp() uint16                 { return m.sp }
func (m CharacterStatistics) Experience() uint32         { return m.experience }
func (m CharacterStatistics) Fame() int16                { return m.fame }
func (m CharacterStatistics) GachaponExperience() uint32 { return m.gachaponExperience }
func (m CharacterStatistics) MapId() uint32              { return m.mapId }
func (m CharacterStatistics) SpawnPoint() byte           { return m.spawnPoint }

func (m CharacterStatistics) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.id)
		WritePaddedString(w, m.name, 13)
		w.WriteByte(m.gender)
		w.WriteByte(m.skinColor)
		w.WriteInt(m.face)
		w.WriteInt(m.hair)

		if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
			for i := 0; i < 3; i++ {
				w.WriteLong(m.petIds[i])
			}
		} else {
			w.WriteLong(m.petIds[0])
		}

		w.WriteByte(m.level)
		w.WriteShort(m.jobId)
		w.WriteShort(m.strength)
		w.WriteShort(m.dexterity)
		w.WriteShort(m.intelligence)
		w.WriteShort(m.luck)
		w.WriteShort(m.hp)
		w.WriteShort(m.maxHp)
		w.WriteShort(m.mp)
		w.WriteShort(m.maxMp)
		w.WriteShort(m.ap)

		if m.hasSPTable {
			// WriteRemainingSkillInfo — currently a stub (no bytes written)
		} else {
			w.WriteShort(m.sp)
		}

		w.WriteInt(m.experience)
		w.WriteInt16(m.fame)

		if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
			w.WriteInt(m.gachaponExperience)
		}

		w.WriteInt(m.mapId)
		w.WriteByte(m.spawnPoint)

		if t.Region() == "GMS" {
			if t.MajorVersion() > 12 {
				w.WriteInt(0)
			} else {
				w.WriteInt64(0)
				w.WriteInt(0)
				w.WriteInt(0)
			}
			if t.MajorVersion() >= 87 {
				w.WriteShort(0) // nSubJob
			}
		} else if t.Region() == "JMS" {
			w.WriteShort(0)
			w.WriteLong(0)
			w.WriteInt(0)
			w.WriteInt(0)
			w.WriteInt(0)
		}

		return w.Bytes()
	}
}

func (m *CharacterStatistics) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.id = r.ReadUint32()
		m.name = ReadPaddedString(r, 13)
		m.gender = r.ReadByte()
		m.skinColor = r.ReadByte()
		m.face = r.ReadUint32()
		m.hair = r.ReadUint32()

		if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
			for i := 0; i < 3; i++ {
				m.petIds[i] = r.ReadUint64()
			}
		} else {
			m.petIds[0] = r.ReadUint64()
		}

		m.level = r.ReadByte()
		m.jobId = r.ReadUint16()
		m.strength = r.ReadUint16()
		m.dexterity = r.ReadUint16()
		m.intelligence = r.ReadUint16()
		m.luck = r.ReadUint16()
		m.hp = r.ReadUint16()
		m.maxHp = r.ReadUint16()
		m.mp = r.ReadUint16()
		m.maxMp = r.ReadUint16()
		m.ap = r.ReadUint16()

		if m.hasSPTable {
			// ReadRemainingSkillInfo — currently a stub (no bytes read)
		} else {
			m.sp = r.ReadUint16()
		}

		m.experience = r.ReadUint32()
		m.fame = r.ReadInt16()

		if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
			m.gachaponExperience = r.ReadUint32()
		}

		m.mapId = r.ReadUint32()
		m.spawnPoint = r.ReadByte()

		if t.Region() == "GMS" {
			if t.MajorVersion() > 12 {
				_ = r.ReadUint32()
			} else {
				_ = r.ReadInt64()
				_ = r.ReadUint32()
				_ = r.ReadUint32()
			}
			if t.MajorVersion() >= 87 {
				_ = r.ReadUint16() // nSubJob
			}
		} else if t.Region() == "JMS" {
			_ = r.ReadUint16()
			_ = r.ReadUint64()
			_ = r.ReadUint32()
			_ = r.ReadUint32()
			_ = r.ReadUint32()
		}
	}
}
