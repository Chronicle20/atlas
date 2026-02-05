package character

import (
	job2 "atlas-character-factory/job"
	"context"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/sirupsen/logrus"
)

func Create(l logrus.FieldLogger) func(ctx context.Context) func(accountId uint32, worldId world.Id, name string, gender byte, mapId _map.Id, jobIndex uint32, subJobIndex uint32, face uint32, hair uint32, hairColor uint32, skinColor byte) (Model, error) {
	return func(ctx context.Context) func(accountId uint32, worldId world.Id, name string, gender byte, mapId _map.Id, jobIndex uint32, subJobIndex uint32, face uint32, hair uint32, hairColor uint32, skinColor byte) (Model, error) {
		return func(accountId uint32, worldId world.Id, name string, gender byte, mapId _map.Id, jobIndex uint32, subJobIndex uint32, face uint32, hair uint32, hairColor uint32, skinColor byte) (Model, error) {
			jobId := job2.JobFromIndex(jobIndex, subJobIndex)

			rm, err := requestCreate(accountId, worldId, name, gender, mapId, jobId, face, hair, hairColor, skinColor)(l, ctx)
			if err != nil {
				return Model{}, err
			}
			return Extract(rm)
		}
	}
}
