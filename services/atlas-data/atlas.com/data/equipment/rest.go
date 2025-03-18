package equipment

import (
	"github.com/jtumidanski/api2go/jsonapi"
	"strconv"
)

type RestModel struct {
	Id            uint32          `json:"-"`
	Strength      uint16          `json:"strength"`
	Dexterity     uint16          `json:"dexterity"`
	Intelligence  uint16          `json:"intelligence"`
	Luck          uint16          `json:"luck"`
	HP            uint16          `json:"hp"`
	MP            uint16          `json:"mp"`
	WeaponAttack  uint16          `json:"weaponAttack"`
	MagicAttack   uint16          `json:"magicAttack"`
	WeaponDefense uint16          `json:"weaponDefense"`
	MagicDefense  uint16          `json:"magicDefense"`
	Accuracy      uint16          `json:"accuracy"`
	Avoidability  uint16          `json:"avoidability"`
	Speed         uint16          `json:"speed"`
	Jump          uint16          `json:"jump"`
	Slots         uint16          `json:"slots"`
	Cash          bool            `json:"cash"`
	EquipSlots    []SlotRestModel `json:"-"`
}

func (r RestModel) GetName() string {
	return "statistics"
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
	rfs := make([]jsonapi.Reference, 0)
	rfs = append(rfs, jsonapi.Reference{Type: "slots", Name: "slots"})
	return rfs
}

func (r RestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	rfs := make([]jsonapi.ReferenceID, 0)
	for _, x := range r.EquipSlots {
		rfs = append(rfs, jsonapi.ReferenceID{
			ID:   x.Id,
			Type: "slots",
			Name: "slots",
		})
	}
	return rfs
}

func (r RestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	rfs := make([]jsonapi.MarshalIdentifier, 0)
	for _, x := range r.EquipSlots {
		rfs = append(rfs, x)
	}
	return rfs
}
func (r *RestModel) SetToOneReferenceID(name string, ID string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(name string, IDs []string) error {
	if name == "slots" {
		res := make([]SlotRestModel, 0)
		for _, x := range IDs {
			rm := SlotRestModel{}
			err := rm.SetID(x)
			if err != nil {
				return err
			}
			res = append(res, rm)
		}
		r.EquipSlots = res
	}
	return nil
}

func (r *RestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	if refMap, ok := references["slots"]; ok {
		res := make([]SlotRestModel, 0)
		for _, rid := range r.GetReferencedIDs() {
			var data jsonapi.Data
			if data, ok = refMap[rid.ID]; ok {
				var rm SlotRestModel
				err := jsonapi.ProcessIncludeData(&rm, data, references)
				if err != nil {
					return err
				}
				_ = rm.SetID(rid.ID)
				res = append(res, rm)
			}
		}
		r.EquipSlots = res
	}
	return nil
}

type SlotRestModel struct {
	Id   string `json:"-"`
	Name string `json:"name"`
	WZ   string `json:"WZ"`
	Slot int16  `json:"slot"`
}

func (r SlotRestModel) GetName() string {
	return "slots"
}

func (r SlotRestModel) GetID() string {
	return r.Id
}

func (r *SlotRestModel) SetID(id string) error {
	r.Id = id
	return nil
}
