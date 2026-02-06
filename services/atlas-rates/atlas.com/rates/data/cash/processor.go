package cash

import (
	"context"

	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

// GetById fetches cash item data from atlas-data by template ID
func GetById(l logrus.FieldLogger) func(ctx context.Context) func(id uint32) (RestModel, error) {
	return func(ctx context.Context) func(id uint32) (RestModel, error) {
		return func(id uint32) (RestModel, error) {
			return requests.Provider[RestModel, RestModel](l, ctx)(requestById(id), func(rm RestModel) (RestModel, error) {
				return rm, nil
			})()
		}
	}
}
