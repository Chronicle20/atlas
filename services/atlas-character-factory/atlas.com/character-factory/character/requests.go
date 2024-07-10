package character

import (
	"atlas-character-factory/inventory/item"
	"atlas-character-factory/job"
	"atlas-character-factory/rest"
	"atlas-character-factory/tenant"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"math"
	"os"
)

const (
	charactersResource     = "characters"
	characterResource      = charactersResource + "/%d"
	characterItemsResource = characterResource + "/inventories/%d/items"
)

func getBaseRequest() string {
	return os.Getenv("CHARACTER_SERVICE_URL")
}

func requestById(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(id uint32) requests.Request[RestModel] {
	return func(id uint32) requests.Request[RestModel] {
		return rest.MakeGetRequest[RestModel](l, span, tenant)(fmt.Sprintf(getBaseRequest()+characterResource, id))
	}
}

func requestCreate(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(accountId uint32, worldId byte, name string, gender byte, jobId job.Id, face uint32, hair uint32, hairColor uint32, skinColor byte) requests.PostRequest[RestModel] {
	return func(accountId uint32, worldId byte, name string, gender byte, jobId job.Id, face uint32, hair uint32, hairColor uint32, skinColor byte) requests.PostRequest[RestModel] {
		i := RestModel{
			AccountId: accountId,
			WorldId:   worldId,
			Name:      name,
			Gender:    gender,
			JobId:     uint16(jobId),
			Face:      face,
			Hair:      hair + hairColor,
			SkinColor: skinColor,
			Level:     1,
		}
		return rest.MakePostRequest[RestModel](l, span, tenant)(fmt.Sprintf(getBaseRequest()+charactersResource), i)
	}
}

func requestCreateItem(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(characterId uint32, itemId uint32) requests.PostRequest[item.RestModel] {
	return func(characterId uint32, itemId uint32) requests.PostRequest[item.RestModel] {
		inventoryType := uint32(math.Floor(float64(itemId) / 1000000))
		i := item.RestModel{ItemId: itemId}
		return rest.MakePostRequest[item.RestModel](l, span, tenant)(fmt.Sprintf(getBaseRequest()+characterItemsResource, characterId, inventoryType), i)
	}
}
