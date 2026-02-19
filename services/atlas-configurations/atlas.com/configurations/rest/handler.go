package rest

import (
	"net/http"

	"github.com/Chronicle20/atlas-rest/server"
	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
)

type HandlerDependency = server.HandlerDependency

type HandlerContext = server.HandlerContext

type GetHandler = server.GetHandler

type InputHandler[M any] = server.InputHandler[M]

func ParseInput[M any](d *HandlerDependency, c *HandlerContext, next InputHandler[M]) http.HandlerFunc {
	return server.ParseInput[M](d, c, next)
}

func RegisterHandler(l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(handlerName string, handler GetHandler) http.HandlerFunc {
	return server.RegisterSimpleHandler(l)
}

func RegisterInputHandler[M any](l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
	return server.RegisterSimpleInputHandler[M](l)
}

func ParseConfigurationType(l logrus.FieldLogger, next func(string) http.HandlerFunc) http.HandlerFunc {
	return server.ParseStringId(l, "type", next)
}

func ParseRegion(l logrus.FieldLogger, next func(string) http.HandlerFunc) http.HandlerFunc {
	return server.ParseStringId(l, "region", next)
}

func ParseMajorVersion(l logrus.FieldLogger, next func(uint16) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint16](l, "majorVersion", next)
}

func ParseMinorVersion(l logrus.FieldLogger, next func(uint16) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint16](l, "minorVersion", next)
}

func ParseTenantId(l logrus.FieldLogger, next func(uuid.UUID) http.HandlerFunc) http.HandlerFunc {
	return server.ParseUUIDId(l, "tenantId", next)
}

func ParseTemplateId(l logrus.FieldLogger, next func(uuid.UUID) http.HandlerFunc) http.HandlerFunc {
	return server.ParseUUIDId(l, "templateId", next)
}

func ParseServiceId(l logrus.FieldLogger, next func(uuid.UUID) http.HandlerFunc) http.HandlerFunc {
	return server.ParseUUIDId(l, "serviceId", next)
}
