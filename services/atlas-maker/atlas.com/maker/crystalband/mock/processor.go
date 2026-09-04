package mock

import (
	"atlas-maker/crystalband"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	GetAllFunc          func() ([]crystalband.Model, error)
	GetAllPagedFunc     func(page model.Page) model.Provider[model.Paged[crystalband.Model]]
	GetByMinLevelFunc   func(minLevel uint32) (crystalband.Model, error)
	CrystalForLevelFunc func(reqLevel uint32) (item.Id, uint32, error)
}

var _ crystalband.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetAll() ([]crystalband.Model, error) {
	if m.GetAllFunc != nil {
		return m.GetAllFunc()
	}
	return nil, nil
}

func (m *ProcessorMock) GetAllPaged(page model.Page) model.Provider[model.Paged[crystalband.Model]] {
	if m.GetAllPagedFunc != nil {
		return m.GetAllPagedFunc(page)
	}
	return model.FixedProvider(model.Paged[crystalband.Model]{})
}

func (m *ProcessorMock) GetByMinLevel(minLevel uint32) (crystalband.Model, error) {
	if m.GetByMinLevelFunc != nil {
		return m.GetByMinLevelFunc(minLevel)
	}
	return crystalband.Model{}, nil
}

func (m *ProcessorMock) CrystalForLevel(reqLevel uint32) (item.Id, uint32, error) {
	if m.CrystalForLevelFunc != nil {
		return m.CrystalForLevelFunc(reqLevel)
	}
	return 0, 0, nil
}
