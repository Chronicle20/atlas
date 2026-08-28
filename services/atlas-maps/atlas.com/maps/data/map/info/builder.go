package info

import (
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

type Builder struct{ m Model }

func NewBuilder() *Builder { return &Builder{} }

func (b *Builder) SetId(v _map.Id) *Builder                { b.m.id = v; return b }
func (b *Builder) SetTimeLimit(v int32) *Builder           { b.m.timeLimit = v; return b }
func (b *Builder) SetForcedReturnMapId(v _map.Id) *Builder { b.m.forcedReturnMapId = v; return b }
func (b *Builder) Build() Model                            { return b.m }
