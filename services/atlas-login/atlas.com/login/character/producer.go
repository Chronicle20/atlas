package character

//import (
//	"atlas-clc/kafka"
//	"github.com/opentracing/opentracing-go"
//	"github.com/sirupsen/logrus"
//)
//
//type characterStatusEvent struct {
//	WorldId     byte   `json:"worldId"`
//	ChannelId   byte   `json:"channelId"`
//	AccountId   uint32 `json:"accountId"`
//	CharacterId uint32 `json:"characterId"`
//	Type        string `json:"type"`
//}
//
//func Logout(l logrus.FieldLogger, span opentracing.Span) func(worldId byte, channelId byte, accountId uint32, characterId uint32) {
//	producer := kafka.ProduceEvent(l, span, "TOPIC_CHARACTER_STATUS")
//	return func(worldId byte, channelId byte, accountId uint32, characterId uint32) {
//		e := &characterStatusEvent{
//			WorldId:     worldId,
//			ChannelId:   channelId,
//			AccountId:   accountId,
//			CharacterId: characterId,
//			Type:        "LOGOUT",
//		}
//		producer(kafka.CreateKey(int(characterId)), e)
//	}
//}
