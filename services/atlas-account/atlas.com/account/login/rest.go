package login

import "strconv"
import "github.com/google/uuid"

type RestModel struct {
	Id        uint32    `json:"-"`
	SessionId uuid.UUID `json:"sessionId"`
	Name      string    `json:"name"`
	Password  string    `json:"password"`
	IpAddress string    `json:"ipAddress"`
	State     int       `json:"state"`
}

func (r RestModel) GetName() string {
	return "login"
}

func (r RestModel) SetID(id string) error {
	nid, err := strconv.Atoi(id)
	if err != nil {
		return err
	}
	r.Id = uint32(nid)
	return nil
}
