package ban

import (
	"strconv"
	"time"
)

type RestModel struct {
	Id         uint32 `json:"-"`
	BanType    byte   `json:"banType"`
	Value      string `json:"value"`
	Reason     string `json:"reason"`
	ReasonCode byte   `json:"reasonCode"`
	Permanent  bool   `json:"permanent"`
	ExpiresAt  time.Time `json:"expiresAt"`
	IssuedBy   string    `json:"issuedBy"`
}

func (r RestModel) GetName() string {
	return "bans"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(idStr string) error {
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:         m.Id(),
		BanType:    byte(m.Type()),
		Value:      m.Value(),
		Reason:     m.Reason(),
		ReasonCode: m.ReasonCode(),
		Permanent:  m.Permanent(),
		ExpiresAt:  m.ExpiresAt(),
		IssuedBy:   m.IssuedBy(),
	}, nil
}

func Extract(rm RestModel) (Model, error) {
	return NewBuilder(
		[16]byte{},
		BanType(rm.BanType),
		rm.Value,
	).
		SetId(rm.Id).
		SetReason(rm.Reason).
		SetReasonCode(rm.ReasonCode).
		SetPermanent(rm.Permanent).
		SetExpiresAt(rm.ExpiresAt).
		SetIssuedBy(rm.IssuedBy).
		Build()
}

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

func TransformCheck(m *Model) CheckRestModel {
	if m == nil {
		return CheckRestModel{
			Banned: false,
		}
	}
	return CheckRestModel{
		Id:         m.Id(),
		Banned:     true,
		BanType:    byte(m.Type()),
		Reason:     m.Reason(),
		ReasonCode: m.ReasonCode(),
		Permanent:  m.Permanent(),
		ExpiresAt:  m.ExpiresAt(),
	}
}
