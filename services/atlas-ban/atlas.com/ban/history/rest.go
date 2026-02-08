package history

import (
	"strconv"
)

type RestModel struct {
	Id            uint64 `json:"-"`
	AccountId     uint32 `json:"accountId"`
	AccountName   string `json:"accountName"`
	IPAddress     string `json:"ipAddress"`
	HWID          string `json:"hwid"`
	Success       bool   `json:"success"`
	FailureReason string `json:"failureReason,omitempty"`
}

func (r RestModel) GetName() string {
	return "login-history"
}

func (r RestModel) GetID() string {
	return strconv.FormatUint(r.Id, 10)
}

func (r *RestModel) SetID(idStr string) error {
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return err
	}
	r.Id = id
	return nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:            m.Id(),
		AccountId:     m.AccountId(),
		AccountName:   m.AccountName(),
		IPAddress:     m.IPAddress(),
		HWID:          m.HWID(),
		Success:       m.Success(),
		FailureReason: m.FailureReason(),
	}, nil
}
