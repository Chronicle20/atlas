package character

import (
	"atlas-character-factory/equipable"
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
	"strings"
)

const (
	resource                   = "characters"
	characterResource          = resource + "/%d"
	byIdResource               = characterResource + "?include=inventory"
	characterItemsResource     = characterResource + "/inventories/%d/items"
	getItemBySlot              = characterItemsResource + "?slot=%d"
	characterEquipmentResource = characterResource + "/equipment/%s/equipable"
)

func getBaseRequest() string {
	return os.Getenv("CHARACTER_SERVICE_URL")
}

func requestById(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(id uint32) requests.Request[RestModel] {
	return func(id uint32) requests.Request[RestModel] {
		return rest.MakeGetRequest[RestModel](l, span, tenant)(fmt.Sprintf(getBaseRequest()+byIdResource, id))
	}
}

func requestCreate(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(accountId uint32, worldId byte, name string, gender byte, mapId uint32, jobId job.Id, face uint32, hair uint32, hairColor uint32, skinColor byte) requests.Request[RestModel] {
	return func(accountId uint32, worldId byte, name string, gender byte, mapId uint32, jobId job.Id, face uint32, hair uint32, hairColor uint32, skinColor byte) requests.Request[RestModel] {
		i := RestModel{
			AccountId: accountId,
			WorldId:   worldId,
			Name:      name,
			Gender:    gender,
			MapId:     mapId,
			JobId:     uint16(jobId),
			Face:      face,
			Hair:      hair + hairColor,
			SkinColor: skinColor,
			Level:     1,
			Hp:        50,
			MaxHp:     50,
			Mp:        5,
			MaxMp:     5,
		}
		return rest.MakePostRequest[RestModel](l, span, tenant)(fmt.Sprintf(getBaseRequest()+resource), i)
	}
}

func requestCreateItem(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(characterId uint32, itemId uint32) requests.Request[item.RestModel] {
	return func(characterId uint32, itemId uint32) requests.Request[item.RestModel] {
		inventoryType := uint32(math.Floor(float64(itemId) / 1000000))
		i := item.RestModel{ItemId: itemId}
		return rest.MakePostRequest[item.RestModel](l, span, tenant)(fmt.Sprintf(getBaseRequest()+characterItemsResource, characterId, inventoryType), i)
	}
}

func requestEquipItem(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(characterId uint32, slotName string, itemId uint32, slot int16) requests.Request[equipable.RestModel] {
	return func(characterId uint32, slotName string, itemId uint32, slot int16) requests.Request[equipable.RestModel] {
		e := equipable.RestModel{ItemId: itemId, Slot: slot}
		return rest.MakePostRequest[equipable.RestModel](l, span, tenant)(fmt.Sprintf(getBaseRequest()+characterEquipmentResource, characterId, strings.ToLower(slotName)), e)
	}
}

func requestEquipableItemBySlot(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(characterId uint32, slot int16) requests.Request[equipable.RestModel] {
	return func(characterId uint32, slot int16) requests.Request[equipable.RestModel] {
		return rest.MakeGetRequest[equipable.RestModel](l, span, tenant)(fmt.Sprintf(getBaseRequest()+getItemBySlot, characterId, 1, slot))
	}
}

func requestItemBySlot(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(characterId uint32, inventoryType int8, slot int16) requests.Request[item.RestModel] {
	return func(characterId uint32, inventoryType int8, slot int16) requests.Request[item.RestModel] {
		return rest.MakeGetRequest[item.RestModel](l, span, tenant)(fmt.Sprintf(getBaseRequest()+getItemBySlot, characterId, inventoryType, slot))
	}
}
