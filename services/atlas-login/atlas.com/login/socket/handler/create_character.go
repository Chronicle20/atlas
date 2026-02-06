package handler

import (
	"atlas-login/character/factory"
	"atlas-login/session"
	"atlas-login/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CreateCharacterHandle = "CreateCharacterHandle"

func CreateCharacterHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader) {
	t := tenant.MustFromContext(ctx)
	return func(s session.Model, r *request.Reader) {
		name := r.ReadAsciiString()
		var jobIndex uint32

		if t.Region() == "GMS" && t.MajorVersion() >= 73 {
			jobIndex = r.ReadUint32()
		} else if t.Region() == "JMS" {
			jobIndex = r.ReadUint32()
		} else {
			jobIndex = 1
		}

		var subJobIndex uint16
		if t.Region() == "GMS" && t.MajorVersion() <= 83 {
			subJobIndex = 0
		} else {
			subJobIndex = r.ReadUint16()
		}
		face := r.ReadUint32()
		hair := r.ReadUint32()

		var hairColor uint32
		var skinColor uint32
		if t.Region() == "JMS" {
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
		if (t.Region() == "GMS" && t.MajorVersion() <= 28) || t.Region() == "JMS" {
			// TODO see if this is just an assumption of if they default to account gender.
			gender = 0
		} else {
			gender = r.ReadByte()
		}

		var strength byte
		var dexterity byte
		var intelligence byte
		var luck byte

		if t.Region() == "GMS" && t.MajorVersion() <= 28 {
			strength = r.ReadByte()
			dexterity = r.ReadByte()
			intelligence = r.ReadByte()
			luck = r.ReadByte()
		} else {
			strength = 13
			dexterity = 4
			intelligence = 4
			luck = 4
		}

		err := factory.NewProcessor(l, ctx).SeedCharacter(s.AccountId(), s.WorldId(), name, jobIndex, subJobIndex, face, hair, hairColor, skinColor, gender, top, bottom, shoes, weapon, strength, dexterity, intelligence, luck)
		if err != nil {
			l.WithError(err).Errorf("Error creating character from seed.")
			err = session.Announce(l)(wp)(writer.AddCharacterEntry)(s, writer.AddCharacterErrorBody(l, t)(writer.AddCharacterCodeUnknownError))
			if err != nil {
				l.WithError(err).Errorf("Unable to show newly created character.")
			}
			return
		}
	}
}
