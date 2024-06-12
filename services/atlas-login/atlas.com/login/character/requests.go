package character

import (
	"os"
)

const (
	Resource = "characters/"
	Seeds    = Resource + "seeds"
)

func getBaseRequest() string {
	return os.Getenv("CHARACTER_SERVICE_URL")
}

//func seedCharacter(l logrus.FieldLogger, span opentracing.Span) func(accountId uint32, worldId byte, name string, job uint32, face uint32, hair uint32, color uint32, skinColor uint32, gender byte, top uint32, bottom uint32, shoes uint32, weapon uint32) (requests.DataBody[properties.Attributes], error) {
//	return func(accountId uint32, worldId byte, name string, job uint32, face uint32, hair uint32, color uint32, skinColor uint32, gender byte, top uint32, bottom uint32, shoes uint32, weapon uint32) (requests.DataBody[properties.Attributes], error) {
//		i := seedInputDataContainer{
//			Data: seedDataBody{
//				Id:   "0",
//				Type: "com.atlas.cos.rest.attribute.CharacterSeedAttributes",
//				Attributes: seedAttributes{
//					AccountId: accountId,
//					WorldId:   worldId,
//					Name:      name,
//					JobIndex:  job,
//					Face:      face,
//					Hair:      hair,
//					HairColor: color,
//					Skin:      skinColor,
//					Gender:    gender,
//					Top:       top,
//					Bottom:    bottom,
//					Shoes:     shoes,
//					Weapon:    weapon,
//				},
//			},
//		}
//		r, _, err := requests.MakePostRequest[properties.Attributes](Seeds, i)(l, span)
//		return r.Data(), err
//	}
//}
