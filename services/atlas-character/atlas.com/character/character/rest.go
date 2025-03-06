package character

import (
	"atlas-character/equipment"
	"atlas-character/equipment/slot"
	"atlas-character/inventory"
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

func (r RestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	var result []jsonapi.ReferenceID
	for _, eid := range slot.Types {
		result = append(result, jsonapi.ReferenceID{
			ID:   fmt.Sprintf("%d-%s", r.Id, eid),
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

	for _, t := range slot.Types {
		if val, ok := r.Equipment[t]; ok {
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
			rm := slot.RestModel{Type: id}
			r.Equipment[slot.Type(id)] = rm
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

func Transform(m Model) (RestModel, error) {
	td := GetTemporalRegistry().GetById(m.Id())

	eqp := make(map[slot.Type]slot.RestModel)
	for t, e := range m.GetEquipment().Slots() {
		erm, err := slot.Transform(e)
		if err != nil {
			return RestModel{}, err
		}
		err = erm.SetID(fmt.Sprintf("%d-%s", m.Id(), t))
		if err != nil {
			return RestModel{}, err
		}
		eqp[t] = erm
	}

	inv, err := inventory.Transform(m.Id())(m.GetInventory())
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
	eqp := equipment.NewModel()
	for t, erm := range m.Equipment {
		e, err := slot.Extract(erm)
		if err != nil {
			return Model{}, err
		}
		eqp.Set(t, e)
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
