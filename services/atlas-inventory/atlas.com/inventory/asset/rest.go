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

type BaseData struct {
	OwnerId uint32 `json:"ownerId"`
}
type StatisticRestData struct {
	Strength      uint16 `json:"strength"`
	Dexterity     uint16 `json:"dexterity"`
	Intelligence  uint16 `json:"intelligence"`
	Luck          uint16 `json:"luck"`
	Hp            uint16 `json:"hp"`
	Mp            uint16 `json:"mp"`
	WeaponAttack  uint16 `json:"weaponAttack"`
	MagicAttack   uint16 `json:"magicAttack"`
	WeaponDefense uint16 `json:"weaponDefense"`
	MagicDefense  uint16 `json:"magicDefense"`
	Accuracy      uint16 `json:"accuracy"`
	Avoidability  uint16 `json:"avoidability"`
	Hands         uint16 `json:"hands"`
	Speed         uint16 `json:"speed"`
	Jump          uint16 `json:"jump"`
}

type CashBaseRestData struct {
	CashId int64 `json:"cashId,string"`
}

type StackableRestData struct {
	Quantity uint32 `json:"quantity"`
}

type EquipableRestData struct {
	BaseData
	StatisticRestData
	Slots          uint16    `json:"slots"`
	Locked         bool      `json:"locked"`
	Spikes         bool      `json:"spikes"`
	KarmaUsed      bool      `json:"karmaUsed"`
	Cold           bool      `json:"cold"`
	CanBeTraded    bool      `json:"canBeTraded"`
	LevelType      byte      `json:"levelType"`
	Level          byte      `json:"level"`
	Experience     uint32    `json:"experience"`
	HammersApplied uint32    `json:"hammersApplied"`
	CreatedAt      time.Time `json:"createdAt"`
}

type CashEquipableRestData struct {
	CashBaseRestData
	BaseData
	StatisticRestData
	Slots          uint16 `json:"slots"`
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
	BaseData
	StackableRestData
	Flag         uint16 `json:"flag"`
	Rechargeable uint64 `json:"rechargeable"`
}

type SetupRestData struct {
	BaseData
	StackableRestData
	Flag uint16 `json:"flag"`
}

type EtcRestData struct {
	BaseData
	StackableRestData
	Flag uint16 `json:"flag"`
}

type CashRestData struct {
	BaseData
	CashBaseRestData
	StackableRestData
	Flag        uint16    `json:"flag"`
	PurchasedBy uint32    `json:"purchasedBy"`
	CreatedAt   time.Time `json:"createdAt"`
}

type PetRestData struct {
	BaseData
	CashBaseRestData
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
	if ReferenceType(temp.ReferenceType) == ReferenceTypeCashEquipable {
		var rd CashEquipableRestData
		if err := json.Unmarshal(temp.ReferenceData, &rd); err != nil {
			return fmt.Errorf("error unmarshaling %s referenceData: %w", ReferenceTypeCashEquipable, err)
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
				BaseData: BaseData{
					OwnerId: em.ownerId,
				},
				StatisticRestData: StatisticRestData{
					Strength:      em.strength,
					Dexterity:     em.dexterity,
					Intelligence:  em.intelligence,
					Luck:          em.luck,
					Hp:            em.hp,
					Mp:            em.mp,
					WeaponAttack:  em.weaponAttack,
					MagicAttack:   em.magicAttack,
					WeaponDefense: em.weaponDefense,
					MagicDefense:  em.magicDefense,
					Accuracy:      em.accuracy,
					Avoidability:  em.avoidability,
					Hands:         em.hands,
					Speed:         em.speed,
					Jump:          em.jump,
				},
				Slots:          em.slots,
				Locked:         em.locked,
				Spikes:         em.spikes,
				KarmaUsed:      em.karmaUsed,
				Cold:           em.cold,
				CanBeTraded:    em.canBeTraded,
				LevelType:      em.levelType,
				Level:          em.level,
				Experience:     em.experience,
				HammersApplied: em.hammersApplied,
				CreatedAt:      em.createdAt,
			}
		}
	}
	if m.ReferenceType() == ReferenceTypeCashEquipable {
		if cem, ok := m.referenceData.(CashEquipableReferenceData); ok {
			brm.ReferenceData = CashEquipableRestData{
				CashBaseRestData: CashBaseRestData{
					CashId: cem.cashId,
				},
			}
		}
	}
	if m.ReferenceType() == ReferenceTypeConsumable {
		if cm, ok := m.referenceData.(ConsumableReferenceData); ok {
			brm.ReferenceData = ConsumableRestData{
				BaseData: BaseData{
					OwnerId: cm.ownerId,
				},
				StackableRestData: StackableRestData{
					Quantity: cm.quantity,
				},
				Flag:         cm.flag,
				Rechargeable: cm.rechargeable,
			}
		}
	}
	if m.ReferenceType() == ReferenceTypeSetup {
		if sm, ok := m.referenceData.(SetupReferenceData); ok {
			brm.ReferenceData = SetupRestData{
				BaseData: BaseData{
					OwnerId: sm.ownerId,
				},
				StackableRestData: StackableRestData{
					Quantity: sm.quantity,
				},
				Flag: sm.flag,
			}
		}
	}
	if m.ReferenceType() == ReferenceTypeEtc {
		if em, ok := m.referenceData.(EtcReferenceData); ok {
			brm.ReferenceData = EtcRestData{
				BaseData: BaseData{
					OwnerId: em.ownerId,
				},
				StackableRestData: StackableRestData{
					Quantity: em.quantity,
				},
				Flag: em.flag,
			}
		}
	}
	if m.ReferenceType() == ReferenceTypeCash {
		if cm, ok := m.referenceData.(CashReferenceData); ok {
			brm.ReferenceData = CashRestData{
				BaseData: BaseData{
					OwnerId: cm.ownerId,
				},
				StackableRestData: StackableRestData{
					Quantity: cm.quantity,
				},
				CashBaseRestData: CashBaseRestData{
					CashId: cm.cashId,
				},
				Flag:        cm.flag,
				PurchasedBy: cm.purchaseBy,
				CreatedAt:   cm.createdAt,
			}
		}
	}
	if m.ReferenceType() == ReferenceTypePet {
		if pm, ok := m.referenceData.(PetReferenceData); ok {
			brm.ReferenceData = PetRestData{
				BaseData: BaseData{
					OwnerId: pm.ownerId,
				},
				CashBaseRestData: CashBaseRestData{
					CashId: pm.cashId,
				},
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
			StatisticData: StatisticData{
				strength:      erm.Strength,
				dexterity:     erm.Dexterity,
				intelligence:  erm.Intelligence,
				luck:          erm.Luck,
				hp:            erm.Hp,
				mp:            erm.Mp,
				weaponAttack:  erm.WeaponAttack,
				magicAttack:   erm.MagicAttack,
				weaponDefense: erm.WeaponDefense,
				magicDefense:  erm.MagicDefense,
				accuracy:      erm.Accuracy,
				avoidability:  erm.Avoidability,
				hands:         erm.Hands,
				speed:         erm.Speed,
				jump:          erm.Jump,
			},
			slots: erm.Slots,
			OwnerData: OwnerData{
				ownerId: erm.OwnerId,
			},
			locked:         erm.Locked,
			spikes:         erm.Spikes,
			karmaUsed:      erm.KarmaUsed,
			cold:           erm.Cold,
			canBeTraded:    erm.CanBeTraded,
			levelType:      erm.LevelType,
			level:          erm.Level,
			experience:     erm.Experience,
			hammersApplied: erm.HammersApplied,
			createdAt:      erm.CreatedAt,
		}
	}
	if cem, ok := rm.ReferenceData.(CashEquipableRestData); ok {
		m.referenceData = CashEquipableReferenceData{
			CashData: CashData{
				cashId: cem.CashId,
			},
		}
	}
	if crm, ok := rm.ReferenceData.(ConsumableRestData); ok {
		m.referenceData = ConsumableReferenceData{
			StackableData: StackableData{
				quantity: crm.Quantity,
			},
			OwnerData: OwnerData{
				ownerId: crm.OwnerId,
			},
			FlagData: FlagData{
				flag: crm.Flag,
			},
			rechargeable: crm.Rechargeable,
		}
	}
	if srm, ok := rm.ReferenceData.(SetupRestData); ok {
		m.referenceData = SetupReferenceData{
			StackableData: StackableData{
				quantity: srm.Quantity,
			},
			OwnerData: OwnerData{
				ownerId: srm.OwnerId,
			},
			FlagData: FlagData{
				flag: srm.Flag,
			},
		}
	}
	if erm, ok := rm.ReferenceData.(EtcRestData); ok {
		m.referenceData = EtcReferenceData{
			StackableData: StackableData{
				quantity: erm.Quantity,
			},
			OwnerData: OwnerData{
				ownerId: erm.OwnerId,
			},
			FlagData: FlagData{
				flag: erm.Flag,
			},
		}
	}
	if crm, ok := rm.ReferenceData.(CashRestData); ok {
		m.referenceData = CashReferenceData{
			CashData: CashData{
				cashId: crm.CashId,
			},
			StackableData: StackableData{
				quantity: crm.Quantity,
			},
			OwnerData: OwnerData{
				ownerId: crm.OwnerId,
			},
			FlagData: FlagData{
				flag: crm.Flag,
			},
			PurchaseData: PurchaseData{
				purchaseBy: crm.PurchasedBy,
			},
			createdAt: crm.CreatedAt,
		}
	}
	if prm, ok := rm.ReferenceData.(PetRestData); ok {
		m.referenceData = PetReferenceData{
			CashData: CashData{
				cashId: prm.CashId,
			},
			OwnerData: OwnerData{
				ownerId: prm.OwnerId,
			},
			FlagData: FlagData{
				flag: prm.Flag,
			},
			PurchaseData: PurchaseData{
				purchaseBy: prm.PurchasedBy,
			},
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
