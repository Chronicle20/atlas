package character

import (
	"atlas-login/character/inventory"
	"atlas-login/character/properties"
	"atlas-login/pet"
	"atlas-login/tenant"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func IsValidName(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(name string) (bool, error) {
	return func(name string) (bool, error) {
		//m, err := regexp.MatchString("[a-zA-Z0-9]{3,12}", name)
		//if err != nil {
		//	return false, err
		//}
		//if !m {
		//	return false, nil
		//}
		//
		//_, err = properties.GetByName(l, span, tenant)(name)
		//if err == nil {
		//	return false, nil
		//}
		//
		//if errors.Is(err, requests.NoResultError) {
		//	return true, nil
		//}
		//
		//if err.Error() != "unable to find character by name" {
		//	return false, nil
		//}
		//
		//bn, err := blocked_name.IsBlockedName(l, span)(name)
		//if bn {
		//	return false, err
		//}

		return true, nil
	}
}

func GetForWorld(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(accountId uint32, worldId byte) ([]Model, error) {
	return func(accountId uint32, worldId byte) ([]Model, error) {
		var characters = make([]Model, 0)
		cs, err := properties.GetForWorld(l, span, tenant)(accountId, worldId)
		if err != nil {
			return nil, err
		}
		for _, x := range cs {
			c, err := fromProperties(l, span)(x)
			if err != nil {
				return nil, err
			}
			characters = append(characters, *c)
		}
		return characters, nil
	}
}

func GetById(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(characterId uint32) (*Model, error) {
	return func(characterId uint32) (*Model, error) {
		cs, err := properties.GetById(l, span, tenant)(characterId)
		if err != nil {
			return nil, err
		}

		c, err := fromProperties(l, span)(cs)
		if err != nil {
			return nil, err
		}
		return c, nil
	}
}

func fromProperties(l logrus.FieldLogger, span opentracing.Span) func(data properties.Model) (*Model, error) {
	return func(data properties.Model) (*Model, error) {
		eq, err := inventory.GetEquippedItemsForCharacter(l, span)(data.Id())
		if err != nil {
			return nil, err
		}

		ps, err := pet.GetForCharacter(nil)(data.Id())
		if err != nil {
			return nil, err
		}

		c := NewCharacter(data, eq, ps)
		return &c, nil
	}
}

//func SeedCharacter(l logrus.FieldLogger, span opentracing.Span) func(accountId uint32, worldId byte, name string, job uint32, face uint32, hair uint32, color uint32, skinColor uint32, gender byte, top uint32, bottom uint32, shoes uint32, weapon uint32) (properties.Model, error) {
//	return func(accountId uint32, worldId byte, name string, job uint32, face uint32, hair uint32, color uint32, skinColor uint32, gender byte, top uint32, bottom uint32, shoes uint32, weapon uint32) (properties.Model, error) {
//		ca, err := seedCharacter(l, span)(accountId, worldId, name, job, face, hair, color, skinColor, gender, top, bottom, shoes, weapon)
//		if err != nil {
//			return properties.Model{}, err
//		}
//		p, err := properties.MakeModel(ca)
//		if err != nil {
//			return properties.Model{}, err
//		}
//		return p, nil
//	}
//}
