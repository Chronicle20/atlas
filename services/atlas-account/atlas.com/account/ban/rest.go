package ban

import (
	"strconv"
	"time"
)

type CheckRestModel struct {
	Id         uint32 `json:"-"`
	Banned     bool   `json:"banned"`
	BanType    byte   `json:"banType,omitempty"`
	Reason     string `json:"reason,omitempty"`
	ReasonCode byte   `json:"reasonCode,omitempty"`
	Permanent  bool   `json:"permanent,omitempty"`
	ExpiresAt  time.Time `json:"expiresAt,omitempty"`
}

func (r CheckRestModel) GetName() string {
	return "ban-checks"
}

func (r CheckRestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *CheckRestModel) SetID(idStr string) error {
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}
