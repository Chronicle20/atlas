package mock

import (
	"atlas-data/characters/templates"
	"atlas-data/document"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	RegisterFunc                  func(s *document.Storage[string, templates.RestModel], r model.Provider[[]templates.RestModel]) error
	RegisterCharacterTemplateFunc func(path string) error
}

var _ templates.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) Register(s *document.Storage[string, templates.RestModel], r model.Provider[[]templates.RestModel]) error {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(s, r)
	}
	return nil
}

func (m *ProcessorMock) RegisterCharacterTemplate(path string) error {
	if m.RegisterCharacterTemplateFunc != nil {
		return m.RegisterCharacterTemplateFunc(path)
	}
	return nil
}
