package model

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type Macros struct {
	macros []Macro
}

func NewMacros(macros ...Macro) Macros {
	return Macros{macros: macros}
}

func (m Macros) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(byte(len(m.macros)))
		for _, v := range m.macros {
			w.WriteByteArray(v.Encode(l, ctx)(options))
		}
		return w.Bytes()
	}
}

type Macro struct {
	name     string
	shout    bool
	skillId1 skill.Id
	skillId2 skill.Id
	skillId3 skill.Id
}

func NewMacro(name string, shout bool, skillId1 skill.Id, skillId2 skill.Id, skillId3 skill.Id) Macro {
	return Macro{
		name:     name,
		shout:    shout,
		skillId1: skillId1,
		skillId2: skillId2,
		skillId3: skillId3,
	}
}

func (m Macro) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.name)
		w.WriteBool(m.shout)
		w.WriteInt(uint32(m.skillId1))
		w.WriteInt(uint32(m.skillId2))
		w.WriteInt(uint32(m.skillId3))
		return w.Bytes()
	}
}
