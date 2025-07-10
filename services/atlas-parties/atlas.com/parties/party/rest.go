package party

import (
	"atlas-parties/character"
	"context"
	"strconv"

	"github.com/Chronicle20/atlas-constants/job"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
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

type MemberRestModel struct {
	Id        uint32 `json:"-"`
	Name      string `json:"name"`
	Level     byte   `json:"level"`
	JobId     job.Id `json:"jobId"`
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

func Transform(l logrus.FieldLogger) func(ctx context.Context) func(m Model) (RestModel, error) {
	return func(ctx context.Context) func(m Model) (RestModel, error) {
		return func(m Model) (RestModel, error) {
			rm := RestModel{
				Id:       m.id,
				LeaderId: m.leaderId,
			}

			ms := make([]MemberRestModel, 0)
			for _, mem := range m.members {
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

func TransformMember(l logrus.FieldLogger) func(ctx context.Context) func(memberId uint32) (MemberRestModel, error) {
	return func(ctx context.Context) func(memberId uint32) (MemberRestModel, error) {
		return func(memberId uint32) (MemberRestModel, error) {
			c, err := character.NewProcessor(l, ctx).GetById(memberId)
			if err != nil {
				return MemberRestModel{}, err
			}
			return MemberRestModel{
				Id:        c.Id(),
				Name:      c.Name(),
				Level:     c.Level(),
				JobId:     c.JobId(),
				WorldId:   c.WorldId(),
				ChannelId: c.ChannelId(),
				MapId:     c.MapId(),
				Online:    c.Online(),
			}, nil
		}
	}
}
