package rest

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

type (
	HandlerDependency = server.HandlerDependency
	HandlerContext    = server.HandlerContext
	GetHandler        = server.GetHandler
)

var RegisterHandler = server.RegisterHandler

func ParseRoomId(l logrus.FieldLogger, next func(uuid.UUID) http.HandlerFunc) http.HandlerFunc {
	return server.ParseUUIDId(l, "roomId", next)
}

func ParseEntryId(l logrus.FieldLogger, next func(uuid.UUID) http.HandlerFunc) http.HandlerFunc {
	return server.ParseUUIDId(l, "entryId", next)
}
