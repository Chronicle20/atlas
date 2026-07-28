package mock

import (
	"atlas-asset-expiration/data"
)

type ProcessorMock struct {
	GetReplaceInfoFunc func(templateId uint32) data.ReplaceInfo
}

var _ data.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetReplaceInfo(templateId uint32) data.ReplaceInfo {
	if m.GetReplaceInfoFunc != nil {
		return m.GetReplaceInfoFunc(templateId)
	}
	return data.ReplaceInfo{}
}
