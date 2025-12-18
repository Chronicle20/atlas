package party

import (
	"github.com/jtumidanski/api2go/jsonapi"
	"strconv"
)

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
			r.Members = append(r.Members, MemberRestModel{
				Id:        uint32(id),
				Name:      "",
				Level:     0,
				JobId:     0,
				WorldId:   0,
				ChannelId: 0,
				MapId:     0,
				Online:    false,
			})
		}
	}
	return nil
}

func Extract(rm RestModel) (Model, error) {
	var members = make([]MemberModel, 0)
	// TODO this information is empty and need a second query ....
	for _, m := range rm.Members {
		mm, err := ExtractMember(m)
		if err != nil {
			return Model{}, err
		}
		members = append(members, mm)
	}

	return Model{
		id:       rm.Id,
		leaderId: rm.LeaderId,
		members:  members,
	}, nil
}

func ExtractMember(rm MemberRestModel) (MemberModel, error) {
	return MemberModel{
		id:        rm.Id,
		name:      rm.Name,
		level:     rm.Level,
		jobId:     rm.JobId,
		worldId:   rm.WorldId,
		channelId: rm.ChannelId,
		mapId:     rm.MapId,
		online:    rm.Online,
	}, nil
}

type MemberRestModel struct {
	Id        uint32 `json:"-"`
	Name      string `json:"name"`
	Level     byte   `json:"level"`
	JobId     uint16 `json:"jobId"`
	WorldId   byte   `json:"worldId"`
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
	Online    bool   `json:"online"`
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
