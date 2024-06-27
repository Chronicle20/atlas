package session

import "strconv"
import "github.com/google/uuid"

type InputRestModel struct {
	Id        uint32    `json:"-"`
	SessionId uuid.UUID `json:"sessionId"`
	Name      string    `json:"name"`
	Password  string    `json:"password"`
	IpAddress string    `json:"ipAddress"`
	State     int       `json:"state"`
}

func (r InputRestModel) GetName() string {
	return "sessions"
}

func (r InputRestModel) SetID(id string) error {
	nid, err := strconv.Atoi(id)
	if err != nil {
		return err
	}
	r.Id = uint32(nid)
	return nil
}

type OutputRestModel struct {
	Id     uint32 `json:"-"`
	Code   string `json:"code"`
	Reason byte   `json:"reason"`
	Until  uint64 `json:"until"`
}

func (r OutputRestModel) GetName() string {
	return "sessions"
}

func (r OutputRestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func Transform(model Model) OutputRestModel {
	rm := OutputRestModel{
		Id:     0,
		Code:   model.Code,
		Reason: model.Reason,
		Until:  model.Until,
	}
	return rm
}
