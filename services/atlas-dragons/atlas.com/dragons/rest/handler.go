package rest

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

type HandlerDependency = server.HandlerDependency

type HandlerContext = server.HandlerContext

type GetHandler = server.GetHandler

var RegisterHandler = server.RegisterHandler

func ParseCharacterId(l logrus.FieldLogger, next func(uint32) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint32](l, "characterId", next)
}

func ParseMapId(l logrus.FieldLogger, next func(_map.Id) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[_map.Id](l, "mapId", next)
}

func ParseChannelId(l logrus.FieldLogger, next func(channel.Id) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[channel.Id](l, "channelId", next)
}

func ParseWorldId(l logrus.FieldLogger, next func(world.Id) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[world.Id](l, "worldId", next)
}

func ParseInstanceId(l logrus.FieldLogger, next func(uuid.UUID) http.HandlerFunc) http.HandlerFunc {
	return server.ParseUUIDId(l, "instanceId", next)
}
