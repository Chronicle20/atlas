package mock

import (
	"atlas-channel/portal"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

type ProcessorMock struct {
	EnterFunc          func(f field.Model, portalName string, characterId uint32) error
	WarpFunc           func(f field.Model, characterId uint32, targetMapId _map.Id) error
	WarpToPositionFunc func(f field.Model, characterId uint32, targetMapId _map.Id, x int16, y int16) error
}

var _ portal.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) Enter(f field.Model, portalName string, characterId uint32) error {
	if m.EnterFunc != nil {
		return m.EnterFunc(f, portalName, characterId)
	}
	return nil
}

func (m *ProcessorMock) Warp(f field.Model, characterId uint32, targetMapId _map.Id) error {
	if m.WarpFunc != nil {
		return m.WarpFunc(f, characterId, targetMapId)
	}
	return nil
}

func (m *ProcessorMock) WarpToPosition(f field.Model, characterId uint32, targetMapId _map.Id, x int16, y int16) error {
	if m.WarpToPositionFunc != nil {
		return m.WarpToPositionFunc(f, characterId, targetMapId, x, y)
	}
	return nil
}
