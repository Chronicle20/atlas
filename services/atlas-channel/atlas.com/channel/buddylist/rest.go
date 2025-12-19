package buddylist

import (
	"atlas-channel/buddylist/buddy"
	"github.com/google/uuid"
)

type RestModel struct {
	Id          uuid.UUID         `json:"-"`
	TenantId    uuid.UUID         `json:"-"`
	CharacterId uint32            `json:"characterId"`
	Capacity    byte              `json:"capacity"`
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

func Extract(rm RestModel) (Model, error) {
	buddies := make([]buddy.Model, 0)
	for _, rb := range rm.Buddies {
		b, err := buddy.Extract(rb)
		if err != nil {
			return Model{}, err
		}
		buddies = append(buddies, b)
	}

	return Model{
		tenantId:    rm.TenantId,
		id:          rm.Id,
		characterId: rm.CharacterId,
		capacity:    rm.Capacity,
		buddies:     buddies,
	}, nil
}
