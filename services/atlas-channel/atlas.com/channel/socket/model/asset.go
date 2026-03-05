package model

import (
	"atlas-channel/asset"
	"math"
	"time"

	"github.com/Chronicle20/atlas-constants/item"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type Asset struct {
	zeroPosition bool
	asset        asset.Model
}

func NewAsset(zeroPosition bool, asset asset.Model) Asset {
	return Asset{
		zeroPosition: zeroPosition,
		asset:        asset,
	}
}

func NewAssetWriter(l logrus.FieldLogger, t tenant.Model, options map[string]interface{}, w *response.Writer) func(zeroPosition bool) model.Operator[asset.Model] {
	return func(zeroPosition bool) model.Operator[asset.Model] {
		return func(a asset.Model) error {
			am := NewAsset(zeroPosition, a)
			am.Encode(l, t, options)(w)
			return nil
		}
	}
}

func (m *Asset) Encode(l logrus.FieldLogger, t tenant.Model, options map[string]interface{}) func(w *response.Writer) {
	if m.asset.IsEquipment() && !m.asset.IsCashEquipment() {
		return m.EncodeEquipableInfo(l, t, options)
	}
	if m.asset.IsCashEquipment() {
		return m.EncodeCashEquipableInfo(l, t, options)
	}
	if m.asset.IsConsumable() || m.asset.IsSetup() || m.asset.IsEtc() {
		return m.EncodeStackableInfo(l, t, options)
	}
	if m.asset.IsPet() {
		return m.EncodePetCashItemInfo(l, t, options)
	}
	if m.asset.IsCash() {
		return m.EncodeCashItemInfo(l, t, options)
	}
	l.Fatalf("unknown item type")
	return nil
}

func (m *Asset) EncodeEquipableInfo(_ logrus.FieldLogger, t tenant.Model, _ map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		slot := m.asset.Slot()
		if !m.zeroPosition {
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

		if (t.Region() == "GMS" && t.MajorVersion() > 12) || t.Region() == "JMS" {
			w.WriteByte(1)
		}
		w.WriteInt(m.asset.TemplateId())
		w.WriteBool(false)
		w.WriteInt64(msTime(m.asset.Expiration()))
		w.WriteByte(byte(m.asset.Slots()))
		w.WriteByte(m.asset.Level())
		if t.Region() == "JMS" {
			w.WriteByte(0)
		}
		w.WriteShort(m.asset.Strength())
		w.WriteShort(m.asset.Dexterity())
		w.WriteShort(m.asset.Intelligence())
		w.WriteShort(m.asset.Luck())
		w.WriteShort(m.asset.Hp())
		w.WriteShort(m.asset.Mp())
		w.WriteShort(m.asset.WeaponAttack())
		w.WriteShort(m.asset.MagicAttack())
		w.WriteShort(m.asset.WeaponDefense())
		w.WriteShort(m.asset.MagicDefense())
		w.WriteShort(m.asset.Accuracy())
		w.WriteShort(m.asset.Avoidability())
		w.WriteShort(m.asset.Hands())
		w.WriteShort(m.asset.Speed())
		w.WriteShort(m.asset.Jump())

		if (t.Region() == "GMS" && t.MajorVersion() > 12) || t.Region() == "JMS" {
			w.WriteAsciiString("") // TODO retrieve owner name from id
			w.WriteShort(m.asset.Flag())
		}

		if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
			w.WriteByte(m.asset.LevelType())
			w.WriteByte(m.asset.Level())
			w.WriteInt(m.asset.Experience())
			w.WriteInt(m.asset.HammersApplied())

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
			w.WriteLong(uint64(getTime(-2)))
			w.WriteInt32(-1)
		}
	}
}

func (m *Asset) EncodeCashEquipableInfo(_ logrus.FieldLogger, t tenant.Model, _ map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		slot := m.asset.Slot()
		if !m.zeroPosition {
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

		if (t.Region() == "GMS" && t.MajorVersion() > 12) || t.Region() == "JMS" {
			w.WriteByte(1)
		}
		w.WriteInt(m.asset.TemplateId())
		w.WriteBool(true)
		if true {
			w.WriteInt64(m.asset.CashId())
		}
		w.WriteInt64(msTime(m.asset.Expiration()))
		w.WriteByte(byte(m.asset.Slots()))
		w.WriteByte(m.asset.Level())
		if t.Region() == "JMS" {
			w.WriteByte(0)
		}
		w.WriteShort(m.asset.Strength())
		w.WriteShort(m.asset.Dexterity())
		w.WriteShort(m.asset.Intelligence())
		w.WriteShort(m.asset.Luck())
		w.WriteShort(m.asset.Hp())
		w.WriteShort(m.asset.Mp())
		w.WriteShort(m.asset.WeaponAttack())
		w.WriteShort(m.asset.MagicAttack())
		w.WriteShort(m.asset.WeaponDefense())
		w.WriteShort(m.asset.MagicDefense())
		w.WriteShort(m.asset.Accuracy())
		w.WriteShort(m.asset.Avoidability())
		w.WriteShort(m.asset.Hands())
		w.WriteShort(m.asset.Speed())
		w.WriteShort(m.asset.Jump())

		if (t.Region() == "GMS" && t.MajorVersion() > 12) || t.Region() == "JMS" {
			w.WriteAsciiString("") // TODO retrieve owner name from id
			w.WriteShort(m.asset.Flag())

			if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
				for i := 0; i < 10; i++ {
					w.WriteByte(0x40)
				}
				w.WriteLong(uint64(getTime(-2)))
				w.WriteInt32(-1)
			}
		}
	}
}

func (m *Asset) EncodeStackableInfo(_ logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		if !m.zeroPosition {
			w.WriteInt8(int8(m.asset.Slot()))
		}
		w.WriteByte(2)
		w.WriteInt(m.asset.TemplateId())
		w.WriteBool(false)
		w.WriteInt64(msTime(m.asset.Expiration()))
		w.WriteShort(uint16(m.asset.Quantity()))
		w.WriteAsciiString("") // TODO
		w.WriteShort(m.asset.Flag())
		if item.IsBullet(item.Id(m.asset.TemplateId())) || item.IsThrowingStar(item.Id(m.asset.TemplateId())) {
			w.WriteLong(m.asset.Rechargeable())
		}
	}
}

func (m *Asset) EncodePetCashItemInfo(_ logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		if !m.zeroPosition {
			w.WriteInt8(int8(m.asset.Slot()))
		}
		w.WriteByte(3)
		w.WriteInt(m.asset.TemplateId())
		w.WriteBool(true)
		w.WriteLong(uint64(m.asset.PetId()))
		w.WriteInt64(msTime(time.Time{}))
		WritePaddedString(w, m.asset.PetName(), 13)
		w.WriteByte(m.asset.PetLevel())
		w.WriteShort(m.asset.Closeness())
		w.WriteByte(m.asset.Fullness())
		w.WriteInt64(msTime(m.asset.Expiration()))
		w.WriteShort(0)   // attribute
		w.WriteShort(0)   // skill
		w.WriteInt(18000) // remaining life
		w.WriteShort(0)   // attribute
		return
	}
}

func (m *Asset) EncodeCashItemInfo(_ logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		if !m.zeroPosition {
			w.WriteInt8(int8(m.asset.Slot()))
		}
		w.WriteByte(2)
		w.WriteInt(m.asset.TemplateId())
		w.WriteBool(true)
		w.WriteInt64(m.asset.CashId())
		w.WriteInt64(msTime(m.asset.Expiration()))
		w.WriteShort(uint16(m.asset.Quantity()))
		w.WriteAsciiString("") // TODO
		w.WriteShort(m.asset.Flag())
		return
	}
}
