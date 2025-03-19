package character

import (
	"atlas-consumables/character/equipment"
	"atlas-consumables/character/equipment/slot"
	"atlas-consumables/character/inventory"
	"fmt"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/jtumidanski/api2go/jsonapi"
	"strconv"
	"strings"
)

type RestModel struct {
	Id                 uint32                       `json:"-"`
	AccountId          uint32                       `json:"accountId"`
	WorldId            byte                         `json:"worldId"`
	Name               string                       `json:"name"`
	Level              byte                         `json:"level"`
	Experience         uint32                       `json:"experience"`
	GachaponExperience uint32                       `json:"gachaponExperience"`
	Strength           uint16                       `json:"strength"`
	Dexterity          uint16                       `json:"dexterity"`
	Intelligence       uint16                       `json:"intelligence"`
	Luck               uint16                       `json:"luck"`
	Hp                 uint16                       `json:"hp"`
	MaxHp              uint16                       `json:"maxHp"`
	Mp                 uint16                       `json:"mp"`
	MaxMp              uint16                       `json:"maxMp"`
	Meso               uint32                       `json:"meso"`
	HpMpUsed           int                          `json:"hpMpUsed"`
	JobId              uint16                       `json:"jobId"`
	SkinColor          byte                         `json:"skinColor"`
	Gender             byte                         `json:"gender"`
	Fame               int16                        `json:"fame"`
	Hair               uint32                       `json:"hair"`
	Face               uint32                       `json:"face"`
	Ap                 uint16                       `json:"ap"`
	Sp                 string                       `json:"sp"`
	MapId              uint32                       `json:"mapId"`
	SpawnPoint         uint32                       `json:"spawnPoint"`
	Gm                 int                          `json:"gm"`
	X                  int16                        `json:"x"`
	Y                  int16                        `json:"y"`
	Stance             byte                         `json:"stance"`
	Equipment          map[slot.Type]slot.RestModel `json:"-"`
	Inventory          inventory.RestModel          `json:"-"`
}

func (r *RestModel) GetName() string {
	return "characters"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(idStr string) error {
	id, err := strconv.Atoi(idStr)
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

func (r RestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	var result []jsonapi.ReferenceID
	for _, s := range slot.Slots {
		result = append(result, jsonapi.ReferenceID{
			ID:   fmt.Sprintf("%d-%s", r.Id, s.Type),
			Type: "equipment",
			Name: "equipment",
		})
	}
	for _, iid := range inventory.Types {
		result = append(result, jsonapi.ReferenceID{
			ID:   fmt.Sprintf("%d-%s", r.Id, iid),
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

	for _, s := range slot.Slots {
		if val, ok := r.Equipment[s.Type]; ok {
			result = append(result, val)
		}
	}
	return result
}

func (r *RestModel) SetToOneReferenceID(name, ID string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(name string, IDs []string) error {
	if name == "equipment" {
		if r.Equipment == nil {
			r.Equipment = make(map[slot.Type]slot.RestModel)
		}

		for _, id := range IDs {
			typ := strings.Split(id, "-")[1]
			rm := slot.RestModel{Type: typ}
			r.Equipment[slot.Type(typ)] = rm
		}
		return nil
	}
	if name == "inventories" {
		for _, id := range IDs {
			typ := strings.Split(id, "-")[1]
			if typ == inventory.TypeEquip {
				r.Inventory.Equipable = inventory.EquipableRestModel{Type: typ}
			}
			if typ == inventory.TypeUse {
				r.Inventory.Useable = inventory.ItemRestModel{Type: typ}
			}
			if typ == inventory.TypeSetup {
				r.Inventory.Setup = inventory.ItemRestModel{Type: typ}
			}
			if typ == inventory.TypeETC {
				r.Inventory.Etc = inventory.ItemRestModel{Type: typ}
			}
			if typ == inventory.TypeCash {
				r.Inventory.Cash = inventory.ItemRestModel{Type: typ}
			}
		}
		return nil
	}
	return nil
}

func (r *RestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	if refMap, ok := references["equipment"]; ok {
		for _, rid := range r.GetReferencedIDs() {
			var data jsonapi.Data
			if data, ok = refMap[rid.ID]; ok {
				typ := slot.Type(strings.Split(rid.ID, "-")[1])
				var srm = r.Equipment[typ]
				err := jsonapi.ProcessIncludeData(&srm, data, references)
				if err != nil {
					return err
				}
				r.Equipment[typ] = srm
			}
		}
	}
	if refMap, ok := references["inventories"]; ok {
		for _, rid := range r.GetReferencedIDs() {
			var data jsonapi.Data
			if data, ok = refMap[rid.ID]; ok {
				typ := strings.Split(rid.ID, "-")[1]
				if typ == inventory.TypeEquip {
					srm := r.Inventory.Equipable
					err := jsonapi.ProcessIncludeData(&srm, data, references)
					if err != nil {
						return err
					}
					r.Inventory.Equipable = srm
					continue
				} else {
					var srm inventory.ItemRestModel
					if typ == inventory.TypeUse {
						srm = r.Inventory.Useable
					}
					if typ == inventory.TypeSetup {
						srm = r.Inventory.Setup
					}
					if typ == inventory.TypeETC {
						srm = r.Inventory.Etc
					}
					if typ == inventory.TypeCash {
						srm = r.Inventory.Cash
					}
					err := jsonapi.ProcessIncludeData(&srm, data, references)
					if err != nil {
						return err
					}
					if typ == inventory.TypeUse {
						r.Inventory.Useable = srm
					}
					if typ == inventory.TypeSetup {
						r.Inventory.Setup = srm
					}
					if typ == inventory.TypeETC {
						r.Inventory.Etc = srm
					}
					if typ == inventory.TypeCash {
						r.Inventory.Cash = srm
					}
				}
			}
		}
	}
	return nil
}

func Extract(rm RestModel) (Model, error) {
	eqp := equipment.NewModel()
	for t, erm := range rm.Equipment {
		e, err := slot.Extract(erm)
		if err != nil {
			return Model{}, err
		}
		eqp.Set(t, e)
	}

	inv, err := model.Map(inventory.Extract)(model.FixedProvider(rm.Inventory))()
	if err != nil {
		return Model{}, err
	}

	return Model{
		id:                 rm.Id,
		accountId:          rm.AccountId,
		worldId:            rm.WorldId,
		name:               rm.Name,
		gender:             rm.Gender,
		skinColor:          rm.SkinColor,
		face:               rm.Face,
		hair:               rm.Hair,
		level:              rm.Level,
		jobId:              rm.JobId,
		strength:           rm.Strength,
		dexterity:          rm.Dexterity,
		intelligence:       rm.Intelligence,
		luck:               rm.Luck,
		hp:                 rm.Hp,
		maxHp:              rm.MaxHp,
		mp:                 rm.Mp,
		maxMp:              rm.MaxMp,
		hpMpUsed:           rm.HpMpUsed,
		ap:                 rm.Ap,
		sp:                 rm.Sp,
		experience:         rm.Experience,
		fame:               rm.Fame,
		gachaponExperience: rm.GachaponExperience,
		mapId:              rm.MapId,
		spawnPoint:         rm.SpawnPoint,
		gm:                 rm.Gm,
		meso:               rm.Meso,
		equipment:          eqp,
		inventory:          inv,
	}, nil
}
