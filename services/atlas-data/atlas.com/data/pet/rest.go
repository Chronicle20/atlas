package pet

import (
	"github.com/Chronicle20/atlas-model/model"
	"github.com/jtumidanski/api2go/jsonapi"
	"strconv"
)

type RestModel struct {
	Id     uint64           `json:"-"`
	Hungry uint32           `json:"hungry"`
	Cash   bool             `json:"cash"`
	Life   uint32           `json:"life"`
	Skills []SkillRestModel `json:"-"`
}

func (r RestModel) GetName() string {
	return "pets"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint64(id)
	return nil
}

func (r RestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{
		{
			Type: "skills",
			Name: "skills",
		},
	}
}

func (r RestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	var result []jsonapi.ReferenceID
	for _, sid := range r.Skills {
		result = append(result, jsonapi.ReferenceID{
			ID:   sid.Id,
			Type: "skills",
			Name: "skills",
		})
	}
	return result
}

func (r RestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	var result []jsonapi.MarshalIdentifier
	for _, s := range r.Skills {
		result = append(result, s)
	}
	return result
}

func (r *RestModel) SetToOneReferenceID(name, ID string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(name string, IDs []string) error {
	if name == "skills" {
		if r.Skills == nil {
			r.Skills = make([]SkillRestModel, 0)
		}

		for _, id := range IDs {
			rm := SkillRestModel{Id: id}
			r.Skills = append(r.Skills, rm)
		}
		return nil
	}
	return nil
}

func (r *RestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	if refMap, ok := references["skills"]; ok {
		res := make([]SkillRestModel, 0)
		for _, srm := range r.Skills {
			var data jsonapi.Data
			if data, ok = refMap[srm.Id]; ok {
				err := jsonapi.ProcessIncludeData(&srm, data, references)
				if err != nil {
					return err
				}
				res = append(res, srm)
			}
		}
		r.Skills = res
	}
	return nil
}

func Transform(m Model) (RestModel, error) {
	srms, err := model.SliceMap(TransformSkill)(model.FixedProvider(m.Skills()))(model.ParallelMap())()
	if err != nil {
		return RestModel{}, err
	}
	return RestModel{
		Id:     m.id,
		Hungry: m.hungry,
		Cash:   m.cash,
		Life:   m.life,
		Skills: srms,
	}, nil
}

type SkillRestModel struct {
	Id          string `json:"-"`
	Increase    uint16 `json:"increase"`
	Probability uint16 `json:"probability"`
}

func (r SkillRestModel) GetName() string {
	return "skills"
}

func (r SkillRestModel) GetID() string {
	return r.Id
}

func (r *SkillRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func TransformSkill(m SkillModel) (SkillRestModel, error) {
	return SkillRestModel{
		Id:          m.id,
		Increase:    m.increase,
		Probability: m.probability,
	}, nil
}
