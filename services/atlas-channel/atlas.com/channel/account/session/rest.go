package session

import (
	"github.com/google/uuid"
	"strconv"
)

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

func (r InputRestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
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

func (r OutputRestModel) SetID(id string) error {
	nid, err := strconv.Atoi(id)
	if err != nil {
		return err
	}
	r.Id = uint32(nid)
	return nil
}
