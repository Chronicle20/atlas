package factory

import (
	"atlas-login/character"
	"atlas-login/tenant"
	"errors"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func SeedCharacter(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(accountId uint32, worldId byte, name string, jobIndex uint32, subJobIndex uint16, face uint32, hair uint32, color uint32, skinColor uint32, gender byte, top uint32, bottom uint32, shoes uint32, weapon uint32) (character.Model, error) {
	return func(accountId uint32, worldId byte, name string, jobIndex uint32, subJobIndex uint16, face uint32, hair uint32, color uint32, skinColor uint32, gender byte, top uint32, bottom uint32, shoes uint32, weapon uint32) (character.Model, error) {
		l.Debugf("Create character [%s] with job [%d:%d] and gender [%d]. Face [%d], hair [%d], hairColor [%d] skinColor [%d], top [%d], bottom [%d], shoes [%d], weapon [%d].", name, jobIndex, subJobIndex, gender, face, hair, color, skinColor, top, bottom, shoes, weapon)
		c, err := requestCreate(l, span, tenant)(accountId, worldId, name, jobIndex, subJobIndex, face, hair, color, skinColor, gender, top, bottom, shoes, weapon)(l)
		if err != nil {
			return character.Model{}, err
		}
		m, err := character.Extract(c)
		if err != nil {
			return character.Model{}, err
		}
		if m.Id() == 0 {
			return character.Model{}, errors.New("unable to create character")
		}
		return m, nil
	}
}
