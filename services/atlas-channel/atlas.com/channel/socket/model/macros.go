package model

import (
	"github.com/Chronicle20/atlas-constants/skill"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type Macros struct {
	macros []Macro
}

func NewMacros(macros ...Macro) Macros {
	return Macros{macros: macros}
}

func (m *Macros) Encode(l logrus.FieldLogger, t tenant.Model, options map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteByte(byte(len(m.macros)))
		for _, v := range m.macros {
			v.Encode(l, t, options)(w)
		}
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

func (m *Macro) Encode(_ logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteAsciiString(m.name)
		w.WriteBool(m.shout)
		w.WriteInt(uint32(m.skillId1))
		w.WriteInt(uint32(m.skillId2))
		w.WriteInt(uint32(m.skillId3))
	}
}
