package mobskill

import (
	"context"

	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func GetByIdAndLevel(l logrus.FieldLogger) func(ctx context.Context) func(skillId uint16, level uint16) (Model, error) {
	return func(ctx context.Context) func(skillId uint16, level uint16) (Model, error) {
		return func(skillId uint16, level uint16) (Model, error) {
			return requests.Provider[RestModel, Model](l, ctx)(requestByIdAndLevel(skillId, level), Extract)()
		}
	}
}
