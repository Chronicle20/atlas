package character

import (
	"atlas-character/equipment"
	"atlas-character/equipment/slot"
	"atlas-character/inventory"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/jtumidanski/api2go/jsonapi"
	"strconv"
)

type RestModel struct {
	Id                 uint32              `json:"-"`
	AccountId          uint32              `json:"accountId"`
	WorldId            byte                `json:"worldId"`
	Name               string              `json:"name"`
	Level              byte                `json:"level"`
	Experience         uint32              `json:"experience"`
	GachaponExperience uint32              `json:"gachaponExperience"`
	Strength           uint16              `json:"strength"`
	Dexterity          uint16              `json:"dexterity"`
	Intelligence       uint16              `json:"intelligence"`
	Luck               uint16              `json:"luck"`
	Hp                 uint16              `json:"hp"`
	MaxHp              uint16              `json:"maxHp"`
	Mp                 uint16              `json:"mp"`
	MaxMp              uint16              `json:"maxMp"`
	Meso               uint32              `json:"meso"`
	HpMpUsed           int                 `json:"hpMpUsed"`
	JobId              uint16              `json:"jobId"`
	SkinColor          byte                `json:"skinColor"`
	Gender             byte                `json:"gender"`
	Fame               int16               `json:"fame"`
	Hair               uint32              `json:"hair"`
	Face               uint32              `json:"face"`
	Ap                 uint16              `json:"ap"`
	Sp                 string              `json:"sp"`
	MapId              uint32              `json:"mapId"`
	SpawnPoint         uint32              `json:"spawnPoint"`
	Gm                 int                 `json:"gm"`
	X                  int16               `json:"x"`
	Y                  int16               `json:"y"`
	Stance             byte                `json:"stance"`
	Equipment          equipment.RestModel `json:"-"`
	Inventory          inventory.RestModel `json:"-"`
}

func (r RestModel) GetName() string {
	return "characters"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func (r RestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{
		{
			Type: "equipment",
			Name: "equipment",
		},
		{
			Type: "inventories",
			Name: "inventories",
		},
	}
}

var equipmentIds = []string{"hat", "medal", "forehead", "ring1", "ring2", "eye", "earring", "shoulder", "cape", "top", "pendant", "weapon", "shield", "gloves", "bottom", "belt", "ring3", "ring4", "shoes"}
var inventoryIds = []string{inventory.TypeEquip, inventory.TypeUse, inventory.TypeSetup, inventory.TypeETC, inventory.TypeCash}

func (r RestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	var result []jsonapi.ReferenceID
	for _, eid := range equipmentIds {
		result = append(result, jsonapi.ReferenceID{
			ID:   eid,
			Type: "equipment",
			Name: "equipment",
		})
	}
	for _, iid := range inventoryIds {
		result = append(result, jsonapi.ReferenceID{
			ID:   iid,
			Type: "inventories",
			Name: "inventories",
		})
	}
	return result
}

func (r RestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	var result []jsonapi.MarshalIdentifier
	result = append(result, r.Inventory.Equipable)
	result = append(result, r.Inventory.Useable)
	result = append(result, r.Inventory.Setup)
	result = append(result, r.Inventory.Etc)
	result = append(result, r.Inventory.Cash)

	result = append(result, r.Equipment.Hat)
	result = append(result, r.Equipment.Medal)
	result = append(result, r.Equipment.Forehead)
	result = append(result, r.Equipment.Ring1)
	result = append(result, r.Equipment.Ring2)
	result = append(result, r.Equipment.Eye)
	result = append(result, r.Equipment.Earring)
	result = append(result, r.Equipment.Shoulder)
	result = append(result, r.Equipment.Cape)
	result = append(result, r.Equipment.Top)
	result = append(result, r.Equipment.Pendant)
	result = append(result, r.Equipment.Weapon)
	result = append(result, r.Equipment.Shield)
	result = append(result, r.Equipment.Gloves)
	result = append(result, r.Equipment.Bottom)
	result = append(result, r.Equipment.Belt)
	result = append(result, r.Equipment.Ring3)
	result = append(result, r.Equipment.Ring4)
	result = append(result, r.Equipment.Shoes)

	return result
}

func (r *RestModel) SetToOneReferenceID(name, ID string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(name string, IDs []string) error {
	if name == "equipment" {
		for _, id := range IDs {
			rm := slot.RestModel{Type: id}
			if id == slot.TypeHat {
				r.Equipment.Hat = rm
			}
			if id == slot.TypeMedal {
				r.Equipment.Medal = rm
			}
			if id == slot.TypeForehead {
				r.Equipment.Forehead = rm
			}
			if id == slot.TypeRing1 {
				r.Equipment.Ring1 = rm
			}
			if id == slot.TypeRing2 {
				r.Equipment.Ring2 = rm
			}
			if id == slot.TypeEye {
				r.Equipment.Eye = rm
			}
			if id == slot.TypeEarring {
				r.Equipment.Earring = rm
			}
			if id == slot.TypeShoulder {
				r.Equipment.Shoulder = rm
			}
			if id == slot.TypeCape {
				r.Equipment.Cape = rm
			}
			if id == slot.TypeTop {
				r.Equipment.Top = rm
			}
			if id == slot.TypePendant {
				r.Equipment.Pendant = rm
			}
			if id == slot.TypeWeapon {
				r.Equipment.Weapon = rm
			}
			if id == slot.TypeShield {
				r.Equipment.Shield = rm
			}
			if id == slot.TypeGloves {
				r.Equipment.Gloves = rm
			}
			if id == slot.TypeBottom {
				r.Equipment.Bottom = rm
			}
			if id == slot.TypeBelt {
				r.Equipment.Belt = rm
			}
			if id == slot.TypeRing3 {
				r.Equipment.Ring3 = rm
			}
			if id == slot.TypeRing4 {
				r.Equipment.Ring4 = rm
			}
			if id == slot.TypeShoes {
				r.Equipment.Shoes = rm
			}
		}
		return nil
	}
	if name == "inventories" {
		for _, id := range IDs {
			if id == inventory.TypeEquip {
				r.Inventory.Equipable = inventory.EquipableRestModel{Type: id}
			}
			if id == inventory.TypeUse {
				r.Inventory.Useable = inventory.ItemRestModel{Type: id}
			}
			if id == inventory.TypeSetup {
				r.Inventory.Setup = inventory.ItemRestModel{Type: id}
			}
			if id == inventory.TypeETC {
				r.Inventory.Etc = inventory.ItemRestModel{Type: id}
			}
			if id == inventory.TypeCash {
				r.Inventory.Cash = inventory.ItemRestModel{Type: id}
			}
		}
		return nil
	}
	return nil
}

func (r *RestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	if refMap, ok := references["equipment"]; ok {
		for _, id := range equipmentIds {
			var data jsonapi.Data
			if data, ok = refMap[id]; ok {
				var srm slot.RestModel
				if id == slot.TypeHat {
					srm = r.Equipment.Hat
				}
				if id == slot.TypeMedal {
					srm = r.Equipment.Medal
				}
				if id == slot.TypeForehead {
					srm = r.Equipment.Forehead
				}
				if id == slot.TypeRing1 {
					srm = r.Equipment.Ring1
				}
				if id == slot.TypeRing2 {
					srm = r.Equipment.Ring2
				}
				if id == slot.TypeEye {
					srm = r.Equipment.Eye
				}
				if id == slot.TypeEarring {
					srm = r.Equipment.Earring
				}
				if id == slot.TypeShoulder {
					srm = r.Equipment.Shoulder
				}
				if id == slot.TypeCape {
					srm = r.Equipment.Cape
				}
				if id == slot.TypeTop {
					srm = r.Equipment.Top
				}
				if id == slot.TypePendant {
					srm = r.Equipment.Pendant
				}
				if id == slot.TypeWeapon {
					srm = r.Equipment.Weapon
				}
				if id == slot.TypeShield {
					srm = r.Equipment.Shield
				}
				if id == slot.TypeGloves {
					srm = r.Equipment.Gloves
				}
				if id == slot.TypeBelt {
					srm = r.Equipment.Belt
				}
				if id == slot.TypeRing3 {
					srm = r.Equipment.Ring3
				}
				if id == slot.TypeRing4 {
					srm = r.Equipment.Ring4
				}
				if id == slot.TypeShoes {
					srm = r.Equipment.Shoes
				}

				err := jsonapi.ProcessIncludeData(&srm, data, references)
				if err != nil {
					return err
				}

				if id == slot.TypeHat {
					r.Equipment.Hat = srm
				}
				if id == slot.TypeMedal {
					r.Equipment.Medal = srm
				}
				if id == slot.TypeForehead {
					r.Equipment.Forehead = srm
				}
				if id == slot.TypeRing1 {
					r.Equipment.Ring1 = srm
				}
				if id == slot.TypeRing2 {
					r.Equipment.Ring2 = srm
				}
				if id == slot.TypeEye {
					r.Equipment.Eye = srm
				}
				if id == slot.TypeEarring {
					r.Equipment.Earring = srm
				}
				if id == slot.TypeShoulder {
					r.Equipment.Shoulder = srm
				}
				if id == slot.TypeCape {
					r.Equipment.Cape = srm
				}
				if id == slot.TypeTop {
					r.Equipment.Top = srm
				}
				if id == slot.TypePendant {
					r.Equipment.Pendant = srm
				}
				if id == slot.TypeWeapon {
					r.Equipment.Weapon = srm
				}
				if id == slot.TypeShield {
					r.Equipment.Shield = srm
				}
				if id == slot.TypeGloves {
					r.Equipment.Gloves = srm
				}
				if id == slot.TypeBelt {
					r.Equipment.Belt = srm
				}
				if id == slot.TypeRing3 {
					r.Equipment.Ring3 = srm
				}
				if id == slot.TypeRing4 {
					r.Equipment.Ring4 = srm
				}
				if id == slot.TypeShoes {
					r.Equipment.Shoes = srm
				}
			}
		}
	}
	if refMap, ok := references["inventories"]; ok {
		for _, id := range inventoryIds {
			var data jsonapi.Data
			if data, ok = refMap[id]; ok {
				if id == inventory.TypeEquip {
					srm := r.Inventory.Equipable
					err := jsonapi.ProcessIncludeData(&srm, data, references)
					if err != nil {
						return err
					}
					r.Inventory.Equipable = srm
					continue
				} else {
					var srm inventory.ItemRestModel
					if id == inventory.TypeUse {
						srm = r.Inventory.Useable
					}
					if id == inventory.TypeSetup {
						srm = r.Inventory.Setup
					}
					if id == inventory.TypeETC {
						srm = r.Inventory.Etc
					}
					if id == inventory.TypeCash {
						srm = r.Inventory.Cash
					}
					err := jsonapi.ProcessIncludeData(&srm, data, references)
					if err != nil {
						return err
					}
					if id == inventory.TypeUse {
						r.Inventory.Useable = srm
					}
					if id == inventory.TypeSetup {
						r.Inventory.Setup = srm
					}
					if id == inventory.TypeETC {
						r.Inventory.Etc = srm
					}
					if id == inventory.TypeCash {
						r.Inventory.Cash = srm
					}
				}
			}
		}
	}
	return nil
}

func Transform(m Model) (RestModel, error) {
	td := GetTemporalRegistry().GetById(m.Id())

	eqp, err := equipment.Transform(m.equipment)
	if err != nil {
		return RestModel{}, err
	}
	inv, err := inventory.Transform(m.inventory)
	if err != nil {
		return RestModel{}, err
	}

	rm := RestModel{
		Id:                 m.id,
		AccountId:          m.accountId,
		WorldId:            m.worldId,
		Name:               m.name,
		Level:              m.level,
		Experience:         m.experience,
		GachaponExperience: m.gachaponExperience,
		Strength:           m.strength,
		Dexterity:          m.dexterity,
		Intelligence:       m.intelligence,
		Luck:               m.luck,
		Hp:                 m.hp,
		MaxHp:              m.maxHp,
		Mp:                 m.mp,
		MaxMp:              m.maxMp,
		Meso:               m.meso,
		HpMpUsed:           m.hpMpUsed,
		JobId:              m.jobId,
		SkinColor:          m.skinColor,
		Gender:             m.gender,
		Fame:               m.fame,
		Hair:               m.hair,
		Face:               m.face,
		Ap:                 m.ap,
		Sp:                 m.sp,
		MapId:              m.mapId,
		SpawnPoint:         m.spawnPoint,
		Gm:                 m.gm,
		X:                  td.X(),
		Y:                  td.Y(),
		Stance:             td.Stance(),
		Equipment:          eqp,
		Inventory:          inv,
	}
	return rm, nil
}

func Extract(m RestModel) (Model, error) {
	eqp, err := model.Map(equipment.Extract)(model.FixedProvider(m.Equipment))()
	if err != nil {
		return Model{}, err
	}

	inv, err := model.Map(inventory.Extract)(model.FixedProvider(m.Inventory))()
	if err != nil {
		return Model{}, err
	}

	return Model{
		id:                 m.Id,
		accountId:          m.AccountId,
		worldId:            m.WorldId,
		name:               m.Name,
		level:              m.Level,
		experience:         m.Experience,
		gachaponExperience: m.GachaponExperience,
		strength:           m.Strength,
		dexterity:          m.Dexterity,
		intelligence:       m.Intelligence,
		luck:               m.Luck,
		hp:                 m.Hp,
		mp:                 m.Mp,
		maxHp:              m.MaxHp,
		maxMp:              m.MaxMp,
		meso:               m.Meso,
		hpMpUsed:           m.HpMpUsed,
		jobId:              m.JobId,
		skinColor:          m.SkinColor,
		gender:             m.Gender,
		fame:               m.Fame,
		hair:               m.Hair,
		face:               m.Face,
		ap:                 m.Ap,
		sp:                 m.Sp,
		mapId:              m.MapId,
		gm:                 m.Gm,
		equipment:          eqp,
		inventory:          inv,
	}, nil
}
