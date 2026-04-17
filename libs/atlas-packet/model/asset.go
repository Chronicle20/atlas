package model

import (
	"context"
	"math"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type Asset struct {
	zeroPosition bool
	slot         int16
	templateId   uint32
	expiration   time.Time
	// equipment fields
	strength       uint16
	dexterity      uint16
	intelligence   uint16
	luck           uint16
	hp             uint16
	mp             uint16
	weaponAttack   uint16
	magicAttack    uint16
	weaponDefense  uint16
	magicDefense   uint16
	accuracy       uint16
	avoidability   uint16
	hands          uint16
	speed          uint16
	jump           uint16
	slots          uint16
	levelType      byte
	level          byte
	experience     uint32
	hammersApplied uint32
	flag           uint16
	// cash fields
	cashId int64
	// stackable fields
	quantity     uint32
	rechargeable uint64
	// pet fields
	petId     uint32
	petName   string
	petLevel  byte
	closeness uint16
	fullness  byte
}

func NewAsset(zeroPosition bool, slot int16, templateId uint32, expiration time.Time) Asset {
	return Asset{
		zeroPosition: zeroPosition,
		slot:         slot,
		templateId:   templateId,
		expiration:   expiration,
	}
}

func (m Asset) ZeroPosition() bool   { return m.zeroPosition }
func (m Asset) Slot() int16          { return m.slot }
func (m Asset) TemplateId() uint32   { return m.templateId }
func (m Asset) Expiration() time.Time { return m.expiration }
func (m Asset) Strength() uint16     { return m.strength }
func (m Asset) Dexterity() uint16    { return m.dexterity }
func (m Asset) Intelligence() uint16 { return m.intelligence }
func (m Asset) Luck() uint16         { return m.luck }
func (m Asset) Hp() uint16           { return m.hp }
func (m Asset) Mp() uint16           { return m.mp }
func (m Asset) WeaponAttack() uint16 { return m.weaponAttack }
func (m Asset) MagicAttack() uint16  { return m.magicAttack }
func (m Asset) WeaponDefense() uint16 { return m.weaponDefense }
func (m Asset) MagicDefense() uint16 { return m.magicDefense }
func (m Asset) Accuracy() uint16     { return m.accuracy }
func (m Asset) Avoidability() uint16 { return m.avoidability }
func (m Asset) Hands() uint16        { return m.hands }
func (m Asset) Speed() uint16        { return m.speed }
func (m Asset) Jump() uint16         { return m.jump }
func (m Asset) Slots() uint16        { return m.slots }
func (m Asset) LevelType() byte      { return m.levelType }
func (m Asset) Level() byte          { return m.level }
func (m Asset) Experience() uint32   { return m.experience }
func (m Asset) HammersApplied() uint32 { return m.hammersApplied }
func (m Asset) Flag() uint16         { return m.flag }
func (m Asset) CashId() int64        { return m.cashId }
func (m Asset) Quantity() uint32     { return m.quantity }
func (m Asset) Rechargeable() uint64 { return m.rechargeable }
func (m Asset) PetId() uint32        { return m.petId }
func (m Asset) PetName() string      { return m.petName }
func (m Asset) PetLevel() byte       { return m.petLevel }
func (m Asset) Closeness() uint16    { return m.closeness }
func (m Asset) Fullness() byte       { return m.fullness }

func (m Asset) inventoryType() inventory.Type {
	t, _ := inventory.TypeFromItemId(item.Id(m.templateId))
	return t
}

func (m Asset) IsEquipment() bool     { return m.inventoryType() == inventory.TypeValueEquip }
func (m Asset) IsCashEquipment() bool { return m.IsEquipment() && m.cashId != 0 }
func (m Asset) IsConsumable() bool    { return m.inventoryType() == inventory.TypeValueUse }
func (m Asset) IsSetup() bool         { return m.inventoryType() == inventory.TypeValueSetup }
func (m Asset) IsEtc() bool           { return m.inventoryType() == inventory.TypeValueETC }
func (m Asset) IsCash() bool          { return m.inventoryType() == inventory.TypeValueCash }
func (m Asset) IsPet() bool           { return m.IsCash() && m.petId > 0 }

// Setters return new Asset (immutable pattern).

func (m Asset) SetEquipmentStats(strength, dexterity, intelligence, luck, hp, mp, weaponAttack, magicAttack, weaponDefense, magicDefense, accuracy, avoidability, hands, speed, jump uint16) Asset {
	m.strength = strength
	m.dexterity = dexterity
	m.intelligence = intelligence
	m.luck = luck
	m.hp = hp
	m.mp = mp
	m.weaponAttack = weaponAttack
	m.magicAttack = magicAttack
	m.weaponDefense = weaponDefense
	m.magicDefense = magicDefense
	m.accuracy = accuracy
	m.avoidability = avoidability
	m.hands = hands
	m.speed = speed
	m.jump = jump
	return m
}

func (m Asset) SetEquipmentMeta(slots uint16, levelType, level byte, experience, hammersApplied uint32, flag uint16) Asset {
	m.slots = slots
	m.levelType = levelType
	m.level = level
	m.experience = experience
	m.hammersApplied = hammersApplied
	m.flag = flag
	return m
}

func (m Asset) SetCashId(cashId int64) Asset {
	m.cashId = cashId
	return m
}

func (m Asset) SetStackableInfo(quantity uint32, flag uint16, rechargeable uint64) Asset {
	m.quantity = quantity
	m.flag = flag
	m.rechargeable = rechargeable
	return m
}

func (m Asset) SetPetInfo(petId uint32, petName string, petLevel, fullness byte, closeness uint16) Asset {
	m.petId = petId
	m.petName = petName
	m.petLevel = petLevel
	m.fullness = fullness
	m.closeness = closeness
	return m
}

func (m *Asset) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	if m.IsEquipment() && !m.IsCashEquipment() {
		return m.encodeEquipableInfo(l, ctx)
	}
	if m.IsCashEquipment() {
		return m.encodeCashEquipableInfo(l, ctx)
	}
	if m.IsConsumable() || m.IsSetup() || m.IsEtc() {
		return m.encodeStackableInfo(l, ctx)
	}
	if m.IsPet() {
		return m.encodePetCashItemInfo(l, ctx)
	}
	if m.IsCash() {
		return m.encodeCashItemInfo(l, ctx)
	}
	l.Fatalf("unknown item type")
	return nil
}

func (m *Asset) encodeEquipableInfo(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		m.encodeSlot(w, t, false)

		if (t.Region() == "GMS" && t.MajorVersion() > 12) || t.Region() == "JMS" {
			w.WriteByte(1)
		}
		w.WriteInt(m.templateId)
		w.WriteBool(false)
		w.WriteInt64(MsTime(m.expiration))
		w.WriteByte(byte(m.slots))
		w.WriteByte(m.level)
		if t.Region() == "JMS" {
			w.WriteByte(0)
		}
		m.encodeEquipmentStats(w)

		if (t.Region() == "GMS" && t.MajorVersion() > 12) || t.Region() == "JMS" {
			w.WriteAsciiString("")
			w.WriteShort(m.flag)
		}

		if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
			w.WriteByte(m.levelType)
			w.WriteByte(m.level)
			w.WriteInt(m.experience)
			w.WriteInt(m.hammersApplied)

			if t.Region() == "JMS" {
				w.WriteByte(0)
				w.WriteShort(0)
				w.WriteShort(0)
				w.WriteShort(0)
				w.WriteShort(0)
				w.WriteShort(0)
				w.WriteInt(0)
			}

			w.WriteLong(0)
			w.WriteInt64(94354848000000000)
			w.WriteInt32(-1)
		}
		return w.Bytes()
	}
}

func (m *Asset) encodeCashEquipableInfo(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		m.encodeSlot(w, t, false)

		if (t.Region() == "GMS" && t.MajorVersion() > 12) || t.Region() == "JMS" {
			w.WriteByte(1)
		}
		w.WriteInt(m.templateId)
		w.WriteBool(true)
		w.WriteInt64(m.cashId)
		w.WriteInt64(MsTime(m.expiration))
		w.WriteByte(byte(m.slots))
		w.WriteByte(m.level)
		if t.Region() == "JMS" {
			w.WriteByte(0)
		}
		m.encodeEquipmentStats(w)

		if (t.Region() == "GMS" && t.MajorVersion() > 12) || t.Region() == "JMS" {
			w.WriteAsciiString("")
			w.WriteShort(m.flag)

			if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
				for i := 0; i < 10; i++ {
					w.WriteByte(0x40)
				}
				w.WriteInt64(94354848000000000)
				w.WriteInt32(-1)
			}
		}
		return w.Bytes()
	}
}

func (m *Asset) encodeStackableInfo(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		if !m.zeroPosition {
			w.WriteInt8(int8(m.slot))
		}
		w.WriteByte(2)
		w.WriteInt(m.templateId)
		w.WriteBool(false)
		w.WriteInt64(MsTime(m.expiration))
		w.WriteShort(uint16(m.quantity))
		w.WriteAsciiString("")
		w.WriteShort(m.flag)
		if item.IsBullet(item.Id(m.templateId)) || item.IsThrowingStar(item.Id(m.templateId)) {
			w.WriteLong(m.rechargeable)
		}
		return w.Bytes()
	}
}

func (m *Asset) encodePetCashItemInfo(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		if !m.zeroPosition {
			w.WriteInt8(int8(m.slot))
		}
		w.WriteByte(3)
		w.WriteInt(m.templateId)
		w.WriteBool(true)
		w.WriteLong(uint64(m.petId))
		w.WriteInt64(MsTime(time.Time{}))
		WritePaddedString(w, m.petName, 13)
		w.WriteByte(m.petLevel)
		w.WriteShort(m.closeness)
		w.WriteByte(m.fullness)
		w.WriteInt64(MsTime(m.expiration))
		w.WriteShort(0)   // attribute
		w.WriteShort(0)   // skill
		w.WriteInt(18000) // remaining life
		w.WriteShort(0)   // attribute
		return w.Bytes()
	}
}

func (m *Asset) encodeCashItemInfo(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		if !m.zeroPosition {
			w.WriteInt8(int8(m.slot))
		}
		w.WriteByte(2)
		w.WriteInt(m.templateId)
		w.WriteBool(true)
		w.WriteInt64(m.cashId)
		w.WriteInt64(MsTime(m.expiration))
		w.WriteShort(uint16(m.quantity))
		w.WriteAsciiString("")
		w.WriteShort(m.flag)
		return w.Bytes()
	}
}

func (m *Asset) encodeSlot(w *response.Writer, t tenant.Model, _ bool) {
	if m.zeroPosition {
		return
	}
	slot := m.slot
	slot = int16(math.Abs(float64(slot)))
	if slot > 100 {
		slot -= 100
	}
	if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
		w.WriteShort(uint16(slot))
	} else {
		w.WriteByte(byte(slot))
	}
}

func (m *Asset) encodeEquipmentStats(w *response.Writer) {
	w.WriteShort(m.strength)
	w.WriteShort(m.dexterity)
	w.WriteShort(m.intelligence)
	w.WriteShort(m.luck)
	w.WriteShort(m.hp)
	w.WriteShort(m.mp)
	w.WriteShort(m.weaponAttack)
	w.WriteShort(m.magicAttack)
	w.WriteShort(m.weaponDefense)
	w.WriteShort(m.magicDefense)
	w.WriteShort(m.accuracy)
	w.WriteShort(m.avoidability)
	w.WriteShort(m.hands)
	w.WriteShort(m.speed)
	w.WriteShort(m.jump)
}

func (m *Asset) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		t := tenant.MustFromContext(ctx)

		var typeByte byte
		if (t.Region() == "GMS" && t.MajorVersion() > 12) || t.Region() == "JMS" {
			typeByte = r.ReadByte()
		} else {
			// For very old versions without a type discriminator, default to equipment.
			typeByte = 1
		}

		m.templateId = r.ReadUint32()
		isCash := r.ReadBool()

		if isCash {
			if typeByte == 3 {
				m.petId = uint32(r.ReadUint64())
			} else {
				m.cashId = r.ReadInt64()
			}
		}

		switch typeByte {
		case 1:
			m.decodeEquipableInfo(r, t, isCash)
		case 2:
			m.decodeStackableInfo(r, isCash)
		case 3:
			m.decodePetInfo(r)
		}
	}
}

func (m *Asset) decodeEquipableInfo(r *request.Reader, t tenant.Model, isCash bool) {
	m.expiration = FromMsTime(r.ReadInt64())
	m.slots = uint16(r.ReadByte())
	m.level = r.ReadByte()
	if t.Region() == "JMS" {
		_ = r.ReadByte()
	}
	m.decodeEquipmentStats(r)

	if (t.Region() == "GMS" && t.MajorVersion() > 12) || t.Region() == "JMS" {
		_ = r.ReadAsciiString() // name
		m.flag = r.ReadUint16()

		if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
			if isCash {
				for i := 0; i < 10; i++ {
					_ = r.ReadByte()
				}
			} else {
				m.levelType = r.ReadByte()
				m.level = r.ReadByte()
				m.experience = r.ReadUint32()
				m.hammersApplied = r.ReadUint32()

				if t.Region() == "JMS" {
					_ = r.ReadByte()
					_ = r.ReadUint16()
					_ = r.ReadUint16()
					_ = r.ReadUint16()
					_ = r.ReadUint16()
					_ = r.ReadUint16()
					_ = r.ReadUint32()
				}

				_ = r.ReadUint64() // 0
			}
			_ = r.ReadInt64() // 94354848000000000
			_ = r.ReadInt32() // -1
		}
	}
}

func (m *Asset) decodeEquipmentStats(r *request.Reader) {
	m.strength = r.ReadUint16()
	m.dexterity = r.ReadUint16()
	m.intelligence = r.ReadUint16()
	m.luck = r.ReadUint16()
	m.hp = r.ReadUint16()
	m.mp = r.ReadUint16()
	m.weaponAttack = r.ReadUint16()
	m.magicAttack = r.ReadUint16()
	m.weaponDefense = r.ReadUint16()
	m.magicDefense = r.ReadUint16()
	m.accuracy = r.ReadUint16()
	m.avoidability = r.ReadUint16()
	m.hands = r.ReadUint16()
	m.speed = r.ReadUint16()
	m.jump = r.ReadUint16()
}

func (m *Asset) decodeStackableInfo(r *request.Reader, isCash bool) {
	m.expiration = FromMsTime(r.ReadInt64())
	m.quantity = uint32(r.ReadUint16())
	_ = r.ReadAsciiString() // ""
	m.flag = r.ReadUint16()
	if !isCash {
		if item.IsBullet(item.Id(m.templateId)) || item.IsThrowingStar(item.Id(m.templateId)) {
			m.rechargeable = r.ReadUint64()
		}
	}
}

func (m *Asset) decodePetInfo(r *request.Reader) {
	_ = FromMsTime(r.ReadInt64()) // msTime(time.Time{})
	m.petName = ReadPaddedString(r, 13)
	m.petLevel = r.ReadByte()
	m.closeness = r.ReadUint16()
	m.fullness = r.ReadByte()
	m.expiration = FromMsTime(r.ReadInt64())
	_ = r.ReadUint16() // attribute
	_ = r.ReadUint16() // skill
	_ = r.ReadUint32() // remaining life
	_ = r.ReadUint16() // attribute
}
