package mock

import (
	"atlas-data/document"
	"atlas-data/quest"
)

type ProcessorMock struct {
	RegisterFunc      func(s *document.Storage[string, quest.RestModel], quest quest.RestModel) error
	RegisterQuestFunc func(path string) error
}

var _ quest.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) Register(s *document.Storage[string, quest.RestModel], q quest.RestModel) error {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(s, q)
	}
	return nil
}

func (m *ProcessorMock) RegisterQuest(path string) error {
	if m.RegisterQuestFunc != nil {
		return m.RegisterQuestFunc(path)
	}
	return nil
}
