package asset

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

type BaseRestModel struct {
	Id            uint32      `json:"-"`
	Slot          int16       `json:"slot"`
	TemplateId    uint32      `json:"templateId"`
	Expiration    time.Time   `json:"expiration"`
	ReferenceId   uint32      `json:"referenceId"`
	ReferenceType string      `json:"referenceType"`
	ReferenceData interface{} `json:"referenceData"`
}

func (r BaseRestModel) GetName() string {
	return "assets"
}

func (r BaseRestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *BaseRestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

type EquipableRestData struct {
	Strength       uint16 `json:"strength"`
	Dexterity      uint16 `json:"dexterity"`
	Intelligence   uint16 `json:"intelligence"`
	Luck           uint16 `json:"luck"`
	HP             uint16 `json:"hp"`
	MP             uint16 `json:"mp"`
	WeaponAttack   uint16 `json:"weaponAttack"`
	MagicAttack    uint16 `json:"magicAttack"`
	WeaponDefense  uint16 `json:"weaponDefense"`
	MagicDefense   uint16 `json:"magicDefense"`
	Accuracy       uint16 `json:"accuracy"`
	Avoidability   uint16 `json:"avoidability"`
	Hands          uint16 `json:"hands"`
	Speed          uint16 `json:"speed"`
	Jump           uint16 `json:"jump"`
	Slots          uint16 `json:"slots"`
	OwnerId        uint32 `json:"ownerId"`
	Locked         bool   `json:"locked"`
	Spikes         bool   `json:"spikes"`
	KarmaUsed      bool   `json:"karmaUsed"`
	Cold           bool   `json:"cold"`
	CanBeTraded    bool   `json:"canBeTraded"`
	LevelType      byte   `json:"levelType"`
	Level          byte   `json:"level"`
	Experience     uint32 `json:"experience"`
	HammersApplied uint32 `json:"hammersApplied"`
}

type ConsumableRestData struct {
	Quantity     uint32 `json:"quantity"`
	OwnerId      uint32 `json:"ownerId"`
	Flag         uint16 `json:"flag"`
	Rechargeable uint64 `json:"rechargeable"`
}

type SetupRestData struct {
	Quantity uint32 `json:"quantity"`
	OwnerId  uint32 `json:"ownerId"`
	Flag     uint16 `json:"flag"`
}

type EtcRestData struct {
	Quantity uint32 `json:"quantity"`
	OwnerId  uint32 `json:"ownerId"`
	Flag     uint16 `json:"flag"`
}

type CashRestData struct {
	CashId      uint64 `json:"cashId"`
	Quantity    uint32 `json:"quantity"`
	OwnerId     uint32 `json:"ownerId"`
	Flag        uint16 `json:"flag"`
	PurchasedBy uint32 `json:"purchasedBy"`
}

type PetRestData struct {
	CashId      uint64 `json:"cashId"`
	OwnerId     uint32 `json:"ownerId"`
	Flag        uint16 `json:"flag"`
	PurchasedBy uint32 `json:"purchasedBy"`
	Name        string `json:"name"`
	Level       byte   `json:"level"`
	Closeness   uint16 `json:"closeness"`
	Fullness    byte   `json:"fullness"`
	Slot        int8   `json:"slot"`
}

func (r *BaseRestModel) UnmarshalJSON(data []byte) error {
	type Alias BaseRestModel
	temp := &struct {
		*Alias
		ReferenceData json.RawMessage `json:"referenceData"`
	}{
		Alias: (*Alias)(r),
	}

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	if ReferenceType(temp.ReferenceType) == ReferenceTypeEquipable {
		var rd EquipableRestData
		if err := json.Unmarshal(temp.ReferenceData, &rd); err != nil {
			return fmt.Errorf("error unmarshaling %s referenceData: %w", ReferenceTypeEquipable, err)
		}
		r.ReferenceData = rd
	}
	if ReferenceType(temp.ReferenceType) == ReferenceTypeConsumable {
		var rd ConsumableRestData
		if err := json.Unmarshal(temp.ReferenceData, &rd); err != nil {
			return fmt.Errorf("error unmarshaling %s referenceData: %w", ReferenceTypeConsumable, err)
		}
		r.ReferenceData = rd
	}
	if ReferenceType(temp.ReferenceType) == ReferenceTypeSetup {
		var rd SetupRestData
		if err := json.Unmarshal(temp.ReferenceData, &rd); err != nil {
			return fmt.Errorf("error unmarshaling %s referenceData: %w", ReferenceTypeSetup, err)
		}
		r.ReferenceData = rd
	}
	if ReferenceType(temp.ReferenceType) == ReferenceTypeEtc {
		var rd EtcRestData
		if err := json.Unmarshal(temp.ReferenceData, &rd); err != nil {
			return fmt.Errorf("error unmarshaling %s referenceData: %w", ReferenceTypeEtc, err)
		}
		r.ReferenceData = rd
	}
	if ReferenceType(temp.ReferenceType) == ReferenceTypeCash {
		var rd CashRestData
		if err := json.Unmarshal(temp.ReferenceData, &rd); err != nil {
			return fmt.Errorf("error unmarshaling %s referenceData: %w", ReferenceTypeCash, err)
		}
		r.ReferenceData = rd
	}
	if ReferenceType(temp.ReferenceType) == ReferenceTypePet {
		var rd PetRestData
		if err := json.Unmarshal(temp.ReferenceData, &rd); err != nil {
			return fmt.Errorf("error unmarshaling %s referenceData: %w", ReferenceTypePet, err)
		}
		r.ReferenceData = rd
	}
	return nil
}

func Transform(m Model[any]) (BaseRestModel, error) {
	brm := BaseRestModel{
		Id:            m.id,
		Slot:          m.slot,
		TemplateId:    m.templateId,
		Expiration:    m.expiration,
		ReferenceId:   m.referenceId,
		ReferenceType: string(m.referenceType),
	}
	if m.ReferenceType() == ReferenceTypeEquipable {
		if em, ok := m.referenceData.(EquipableReferenceData); ok {
			brm.ReferenceData = EquipableRestData{
				Strength:       em.strength,
				Dexterity:      em.dexterity,
				Intelligence:   em.intelligence,
				Luck:           em.luck,
				HP:             em.hp,
				MP:             em.mp,
				WeaponAttack:   em.weaponAttack,
				MagicAttack:    em.magicAttack,
				WeaponDefense:  em.weaponDefense,
				MagicDefense:   em.magicDefense,
				Accuracy:       em.accuracy,
				Avoidability:   em.avoidability,
				Hands:          em.hands,
				Speed:          em.speed,
				Jump:           em.jump,
				Slots:          em.slots,
				OwnerId:        em.ownerId,
				Locked:         em.locked,
				Spikes:         em.spikes,
				KarmaUsed:      em.karmaUsed,
				Cold:           em.cold,
				CanBeTraded:    em.canBeTraded,
				LevelType:      em.levelType,
				Level:          em.level,
				Experience:     em.experience,
				HammersApplied: em.hammersApplied,
			}
		}
	}
	if m.ReferenceType() == ReferenceTypeConsumable {
		if cm, ok := m.referenceData.(ConsumableReferenceData); ok {
			brm.ReferenceData = ConsumableRestData{
				Quantity:     cm.quantity,
				OwnerId:      cm.ownerId,
				Flag:         cm.flag,
				Rechargeable: cm.rechargeable,
			}
		}
	}
	if m.ReferenceType() == ReferenceTypeSetup {
		if sm, ok := m.referenceData.(SetupReferenceData); ok {
			brm.ReferenceData = SetupRestData{
				Quantity: sm.quantity,
				OwnerId:  sm.ownerId,
				Flag:     sm.flag,
			}
		}
	}
	if m.ReferenceType() == ReferenceTypeEtc {
		if em, ok := m.referenceData.(EtcReferenceData); ok {
			brm.ReferenceData = EtcRestData{
				Quantity: em.quantity,
				OwnerId:  em.ownerId,
				Flag:     em.flag,
			}
		}
	}
	if m.ReferenceType() == ReferenceTypeCash {
		if cm, ok := m.referenceData.(CashReferenceData); ok {
			brm.ReferenceData = CashRestData{
				CashId:      cm.cashId,
				Quantity:    cm.quantity,
				OwnerId:     cm.ownerId,
				Flag:        cm.flag,
				PurchasedBy: cm.purchaseBy,
			}
		}
	}
	if m.ReferenceType() == ReferenceTypePet {
		if pm, ok := m.referenceData.(PetReferenceData); ok {
			brm.ReferenceData = PetRestData{
				CashId:      pm.cashId,
				OwnerId:     pm.ownerId,
				Flag:        pm.flag,
				PurchasedBy: pm.purchaseBy,
				Name:        pm.name,
				Level:       pm.level,
				Closeness:   pm.closeness,
				Fullness:    pm.fullness,
				Slot:        pm.slot,
			}
		}
	}
	return brm, nil
}

func Extract(rm BaseRestModel) (Model[any], error) {
	var m Model[any]
	m = Model[any]{
		id:            rm.Id,
		slot:          rm.Slot,
		templateId:    rm.TemplateId,
		expiration:    rm.Expiration,
		referenceId:   rm.ReferenceId,
		referenceType: ReferenceType(rm.ReferenceType),
	}

	if erm, ok := rm.ReferenceData.(EquipableRestData); ok {
		m.referenceData = EquipableReferenceData{
			strength:       erm.Strength,
			dexterity:      erm.Dexterity,
			intelligence:   erm.Intelligence,
			luck:           erm.Luck,
			hp:             erm.HP,
			mp:             erm.MP,
			weaponAttack:   erm.WeaponAttack,
			magicAttack:    erm.MagicAttack,
			weaponDefense:  erm.WeaponDefense,
			magicDefense:   erm.MagicDefense,
			accuracy:       erm.Accuracy,
			avoidability:   erm.Avoidability,
			hands:          erm.Hands,
			speed:          erm.Speed,
			jump:           erm.Jump,
			slots:          erm.Slots,
			ownerId:        erm.OwnerId,
			locked:         erm.Locked,
			spikes:         erm.Spikes,
			karmaUsed:      erm.KarmaUsed,
			cold:           erm.Cold,
			canBeTraded:    erm.CanBeTraded,
			levelType:      erm.LevelType,
			level:          erm.Level,
			experience:     erm.Experience,
			hammersApplied: erm.HammersApplied,
		}
	}
	if crm, ok := rm.ReferenceData.(ConsumableRestData); ok {
		m.referenceData = ConsumableReferenceData{
			quantity:     crm.Quantity,
			ownerId:      crm.OwnerId,
			flag:         crm.Flag,
			rechargeable: crm.Rechargeable,
		}
	}
	if srm, ok := rm.ReferenceData.(SetupRestData); ok {
		m.referenceData = SetupReferenceData{
			quantity: srm.Quantity,
			ownerId:  srm.OwnerId,
			flag:     srm.Flag,
		}
	}
	if erm, ok := rm.ReferenceData.(EtcRestData); ok {
		m.referenceData = EtcReferenceData{
			quantity: erm.Quantity,
			ownerId:  erm.OwnerId,
			flag:     erm.Flag,
		}
	}
	if crm, ok := rm.ReferenceData.(CashRestData); ok {
		m.referenceData = CashReferenceData{
			cashId:     crm.CashId,
			quantity:   crm.Quantity,
			ownerId:    crm.OwnerId,
			flag:       crm.Flag,
			purchaseBy: crm.PurchasedBy,
		}
	}
	if prm, ok := rm.ReferenceData.(PetRestData); ok {
		m.referenceData = PetReferenceData{
			cashId:        prm.CashId,
			ownerId:       prm.OwnerId,
			flag:          prm.Flag,
			purchaseBy:    prm.PurchasedBy,
			name:          prm.Name,
			level:         prm.Level,
			closeness:     prm.Closeness,
			fullness:      prm.Fullness,
			expiration:    rm.Expiration,
			slot:          prm.Slot,
			attribute:     0,
			skill:         0,
			remainingLife: 0,
			attribute2:    0,
		}
	}

	return m, nil
}
