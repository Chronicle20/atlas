package rest

import (
	"net/http"

	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

type HandlerDependency = server.HandlerDependency

type HandlerContext = server.HandlerContext

type GetHandler = server.GetHandler

type InputHandler[M any] = server.InputHandler[M]

var RegisterHandler = server.RegisterHandler

func RegisterInputHandler[M any](l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
	return server.RegisterInputHandler[M](l)
}

// ParseWorldId parses the {worldId} path var into a world.Id (a byte).
func ParseWorldId(l logrus.FieldLogger, next func(worldId world.Id) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[world.Id](l, "worldId", next)
}

// ParseCharacterId parses the {characterId} path var into a uint32.
func ParseCharacterId(l logrus.FieldLogger, next func(characterId uint32) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint32](l, "characterId", next)
}

// ParseAccountId parses the {accountId} path var into a uint32.
func ParseAccountId(l logrus.FieldLogger, next func(accountId uint32) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint32](l, "accountId", next)
}

// ParseListingId parses the {listingId} path var (a UUID string).
func ParseListingId(l logrus.FieldLogger, next func(listingId string) http.HandlerFunc) http.HandlerFunc {
	return server.ParseStringId(l, "listingId", next)
}

// ParseHoldingId parses the {holdingId} path var (a UUID string).
func ParseHoldingId(l logrus.FieldLogger, next func(holdingId string) http.HandlerFunc) http.HandlerFunc {
	return server.ParseStringId(l, "holdingId", next)
}
