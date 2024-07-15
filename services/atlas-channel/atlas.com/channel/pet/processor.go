package pet

import "github.com/sirupsen/logrus"

func GetForCharacter(_ logrus.FieldLogger) func(characterId uint32) ([]Model, error) {
	return func(characterId uint32) ([]Model, error) {
		return make([]Model, 0), nil
	}
}
