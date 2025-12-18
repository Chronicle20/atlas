package invite

import (
	"strconv"
	"time"
)

type RestModel struct {
	Id           uint32    `json:"-"`
	Type         string    `json:"type"`
	ReferenceId  uint32    `json:"referenceId"`
	OriginatorId uint32    `json:"originatorId"`
	TargetId     uint32    `json:"targetId"`
	Age          time.Time `json:"age"`
}

func (r RestModel) GetName() string {
	return "invites"
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

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:           m.id,
		Type:         m.inviteType,
		ReferenceId:  m.referenceId,
		OriginatorId: m.originatorId,
		TargetId:     m.targetId,
		Age:          m.age,
	}, nil
}
