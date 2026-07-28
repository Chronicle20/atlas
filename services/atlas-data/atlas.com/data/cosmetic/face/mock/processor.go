package mock

import (
	"atlas-data/cosmetic/face"
	"atlas-data/document"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	RegisterFunc     func(s *document.Storage[string, face.RestModel], r model.Provider[face.RestModel]) error
	RegisterFaceFunc func(path string) error
}

var _ face.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) Register(s *document.Storage[string, face.RestModel], r model.Provider[face.RestModel]) error {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(s, r)
	}
	return nil
}

func (m *ProcessorMock) RegisterFace(path string) error {
	if m.RegisterFaceFunc != nil {
		return m.RegisterFaceFunc(path)
	}
	return nil
}
