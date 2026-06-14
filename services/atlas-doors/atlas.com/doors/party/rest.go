package party

import (
	"strconv"

	"github.com/jtumidanski/api2go/jsonapi"
)

// RestModel is the JSON:API representation of a party as returned by
// atlas-parties. Members are a to-many relationship stored in join order.
type RestModel struct {
	Id       uint32            `json:"-"`
	LeaderId uint32            `json:"leaderId"`
	Members  []MemberRestModel `json:"-"`
}

func (r RestModel) GetName() string {
	return "parties"
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
			Type: "members",
			Name: "members",
		},
	}
}

func (r RestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	var result []jsonapi.ReferenceID
	for _, v := range r.Members {
		result = append(result, jsonapi.ReferenceID{
			ID:   v.GetID(),
			Type: "members",
			Name: "members",
		})
	}
	return result
}

func (r RestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	var result []jsonapi.MarshalIdentifier
	for key := range r.Members {
		result = append(result, r.Members[key])
	}
	return result
}

func (r *RestModel) SetToManyReferenceIDs(name string, IDs []string) error {
	if name == "members" {
		for _, ID := range IDs {
			id, err := strconv.Atoi(ID)
			if err != nil {
				return err
			}
			r.Members = append(r.Members, MemberRestModel{Id: uint32(id)})
		}
	}
	return nil
}

func (r *RestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	if refMap, ok := references["members"]; ok {
		var nm []MemberRestModel
		for _, m := range r.Members {
			if data, ok := refMap[m.GetID()]; ok {
				srm := MemberRestModel{}
				err := jsonapi.ProcessIncludeData(&srm, data, references)
				if err != nil {
					return err
				}
				err = srm.SetID(m.GetID())
				if err != nil {
					return err
				}
				nm = append(nm, srm)
			}
		}
		r.Members = nm
	}
	return nil
}

// Extract converts a RestModel into a Model. Member order is preserved as
// returned by atlas-parties (join order, leader at index 0). No re-sort is
// applied.
func Extract(rm RestModel) (Model, error) {
	members := make([]uint32, 0, len(rm.Members))
	for _, m := range rm.Members {
		members = append(members, m.Id)
	}
	return Model{
		id:       rm.Id,
		leaderId: rm.LeaderId,
		members:  members,
	}, nil
}

// MemberRestModel is the minimal member resource needed by atlas-doors. Only
// the Id field is used; we preserve the full JSON:API resource shape so that
// SetReferencedStructs can be satisfied by the api2go unmarshaller.
type MemberRestModel struct {
	Id uint32 `json:"-"`
}

func (r MemberRestModel) GetName() string {
	return "members"
}

func (r MemberRestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *MemberRestModel) SetID(idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}
