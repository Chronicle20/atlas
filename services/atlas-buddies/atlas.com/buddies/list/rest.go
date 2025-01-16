package list

import (
	"atlas-buddies/buddy"
	"github.com/google/uuid"
)

type RestModel struct {
	Id          uuid.UUID         `json:"-"`
	TenantId    uuid.UUID         `json:"-"`
	CharacterId uint32            `json:"characterId"`
	Capacity    uint32            `json:"capacity"`
	Buddies     []buddy.RestModel `json:"buddies"`
}

func (r RestModel) GetName() string {
	return "buddy-list"
}

func (r RestModel) GetID() string {
	return r.Id.String()
}

func (r *RestModel) SetID(strId string) error {
	if strId == "" {
		r.Id = uuid.New()
		return nil
	}

	id, err := uuid.Parse(strId)
	if err != nil {
		return err
	}
	r.Id = id
	return nil
}

func Transform(m Model) (RestModel, error) {
	buddies := make([]buddy.RestModel, 0)
	for _, bm := range m.buddies {
		rb, err := buddy.Transform(bm)
		if err != nil {
			return RestModel{}, err
		}
		buddies = append(buddies, rb)
	}

	return RestModel{
		Id:          m.id,
		TenantId:    m.tenantId,
		CharacterId: m.characterId,
		Capacity:    m.capacity,
		Buddies:     buddies,
	}, nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		tenantId:    rm.TenantId,
		id:          rm.Id,
		characterId: rm.CharacterId,
		capacity:    rm.Capacity,
		buddies:     nil,
	}, nil
}
