package info

import (
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

type Model struct {
	id                _map.Id
	timeLimit         int32
	forcedReturnMapId _map.Id
}

func (m Model) Id() _map.Id {
	return m.id
}

func (m Model) TimeLimit() int32 {
	return m.timeLimit
}

func (m Model) ForcedReturnMapId() _map.Id {
	return m.forcedReturnMapId
}

func (m Model) IsTimeLimited() bool {
	return m.timeLimit > 0 && m.forcedReturnMapId != _map.EmptyMapId
}

type Builder struct{ m Model }

func NewBuilder() *Builder { return &Builder{} }

func (b *Builder) SetId(v _map.Id) *Builder                { b.m.id = v; return b }
func (b *Builder) SetTimeLimit(v int32) *Builder           { b.m.timeLimit = v; return b }
func (b *Builder) SetForcedReturnMapId(v _map.Id) *Builder { b.m.forcedReturnMapId = v; return b }
func (b *Builder) Build() Model                            { return b.m }
