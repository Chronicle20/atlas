package mock

import (
	"atlas-account/ban"
)

type ProcessorMock struct {
	CheckBanFunc func(ip string, hwid string, accountId uint32) (ban.CheckRestModel, error)
}

var _ ban.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) CheckBan(ip string, hwid string, accountId uint32) (ban.CheckRestModel, error) {
	if m.CheckBanFunc != nil {
		return m.CheckBanFunc(ip, hwid, accountId)
	}
	return ban.CheckRestModel{}, nil
}
