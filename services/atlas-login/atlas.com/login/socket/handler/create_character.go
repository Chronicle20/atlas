package handler

import (
	"atlas-login/character/factory"
	"atlas-login/session"
	"atlas-login/socket/writer"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

const CreateCharacterHandle = "CreateCharacterHandle"

func CreateCharacterHandleFunc(l logrus.FieldLogger, span opentracing.Span, wp writer.Producer) func(s session.Model, r *request.Reader) {
	addCharacterEntryFunc := session.Announce(l)(wp)(writer.AddCharacterEntry)
	return func(s session.Model, r *request.Reader) {
		name := r.ReadAsciiString()
		var jobIndex uint32
		if s.Tenant().Region == "GMS" && s.Tenant().MajorVersion >= 73 {
			jobIndex = r.ReadUint32()
		} else if s.Tenant().Region == "JMS" {
			jobIndex = r.ReadUint32()
		} else {
			jobIndex = 1
		}

		var subJobIndex uint16
		if s.Tenant().Region == "GMS" && s.Tenant().MajorVersion <= 83 {
			subJobIndex = 0
		} else {
			subJobIndex = r.ReadUint16()
		}
		face := r.ReadUint32()
		hair := r.ReadUint32()

		var hairColor uint32
		var skinColor uint32
		if s.Tenant().Region == "JMS" {
			hairColor = 0
			skinColor = 0
		} else {
			hairColor = r.ReadUint32()
			skinColor = r.ReadUint32()
		}

		top := r.ReadUint32()
		bottom := r.ReadUint32()
		shoes := r.ReadUint32()
		weapon := r.ReadUint32()

		var gender byte
		if (s.Tenant().Region == "GMS" && s.Tenant().MajorVersion <= 28) || s.Tenant().Region == "JMS" {
			// TODO see if this is just an assumption of if they default to account gender.
			gender = 0
		} else {
			gender = r.ReadByte()
		}

		var strength byte
		var dexterity byte
		var intelligence byte
		var luck byte

		if s.Tenant().Region == "GMS" && s.Tenant().MajorVersion <= 28 {
			strength = r.ReadByte()
			dexterity = r.ReadByte()
			intelligence = r.ReadByte()
			luck = r.ReadByte()
		}

		m, err := factory.SeedCharacter(l, span, s.Tenant())(s.AccountId(), s.WorldId(), name, jobIndex, subJobIndex, face, hair, hairColor, skinColor, gender, top, bottom, shoes, weapon, strength, dexterity, intelligence, luck)
		if err != nil {
			l.WithError(err).Errorf("Error creating character from seed.")
			err = addCharacterEntryFunc(s, writer.AddCharacterErrorBody(l, s.Tenant())(writer.AddCharacterCodeUnknownError))
			if err != nil {
				l.WithError(err).Errorf("Unable to show newly created character.")
			}
			return
		}

		err = addCharacterEntryFunc(s, writer.AddCharacterEntryBody(l, s.Tenant())(m))
		if err != nil {
			l.WithError(err).Errorf("Unable to show newly created character.")
		}
	}
}
