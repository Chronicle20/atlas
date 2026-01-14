package messenger

import (
	"atlas-messengers/character"
	"context"
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"strconv"
)

type RestModel struct {
	Id      uint32            `json:"-"`
	Members []MemberRestModel `json:"-"`
}

func (r RestModel) GetName() string {
	return "messengers"
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
				WorldId:   world.Id(0),
				ChannelId: channel.Id(0),
				Slot:      0,
				Online:    false,
			})
		}
	}
	return nil
}

type MemberRestModel struct {
	Id        uint32     `json:"-"`
	Name      string     `json:"name"`
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	Online    bool       `json:"online"`
	Slot      byte       `json:"slot"`
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

func Transform(l logrus.FieldLogger) func(ctx context.Context) func(m Model) (RestModel, error) {
	return func(ctx context.Context) func(m Model) (RestModel, error) {
		return func(m Model) (RestModel, error) {
			rm := RestModel{
				Id: m.Id(),
			}

			ms := make([]MemberRestModel, 0)
			for _, mem := range m.Members() {
				mrm, err := TransformMember(l)(ctx)(mem)
				if err != nil {
					return RestModel{}, err
				}
				ms = append(ms, mrm)
			}
			rm.Members = ms
			return rm, nil
		}
	}
}

func TransformMember(l logrus.FieldLogger) func(ctx context.Context) func(m MemberModel) (MemberRestModel, error) {
	return func(ctx context.Context) func(m MemberModel) (MemberRestModel, error) {
		return func(m MemberModel) (MemberRestModel, error) {
			c, err := character.GetById(l)(ctx)(m.Id())
			if err != nil {
				return MemberRestModel{}, err
			}
			return MemberRestModel{
				Id:        c.Id(),
				Name:      c.Name(),
				WorldId:   c.WorldId(),
				ChannelId: c.ChannelId(),
				Online:    c.Online(),
				Slot:      m.Slot(),
			}, nil
		}
	}
}
