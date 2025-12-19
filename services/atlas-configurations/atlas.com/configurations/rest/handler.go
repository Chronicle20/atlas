package rest

import (
	"context"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"strconv"
)

type HandlerDependency struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func (h HandlerDependency) Logger() logrus.FieldLogger {
	return h.l
}

func (h HandlerDependency) Context() context.Context {
	return h.ctx
}

type HandlerContext struct {
	si jsonapi.ServerInformation
}

func (h HandlerContext) ServerInformation() jsonapi.ServerInformation {
	return h.si
}

type GetHandler func(d *HandlerDependency, c *HandlerContext) http.HandlerFunc

type InputHandler[M any] func(d *HandlerDependency, c *HandlerContext, model M) http.HandlerFunc

func ParseInput[M any](d *HandlerDependency, c *HandlerContext, next InputHandler[M]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var model M

		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		err = jsonapi.Unmarshal(body, &model)
		if err != nil {
			d.l.WithError(err).Errorln("Deserializing input", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(d, c, model)(w, r)
	}
}

func RegisterHandler(l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(handlerName string, handler GetHandler) http.HandlerFunc {
	return func(si jsonapi.ServerInformation) func(handlerName string, handler GetHandler) http.HandlerFunc {
		return func(handlerName string, handler GetHandler) http.HandlerFunc {
			return server.RetrieveSpan(l, handlerName, context.Background(), func(sl logrus.FieldLogger, sctx context.Context) http.HandlerFunc {
				fl := sl.WithFields(logrus.Fields{"originator": handlerName, "type": "rest_handler"})
				return handler(&HandlerDependency{l: fl, ctx: sctx}, &HandlerContext{si: si})
			})
		}
	}
}

func RegisterInputHandler[M any](l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
	return func(si jsonapi.ServerInformation) func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
		return func(handlerName string, handler InputHandler[M]) http.HandlerFunc {
			return server.RetrieveSpan(l, handlerName, context.Background(), func(sl logrus.FieldLogger, sctx context.Context) http.HandlerFunc {
				fl := sl.WithFields(logrus.Fields{"originator": handlerName, "type": "rest_handler"})
				return ParseInput[M](&HandlerDependency{l: fl, ctx: sctx}, &HandlerContext{si: si}, handler)
			})
		}
	}
}

type ConfigurationTypeHandler func(theType string) http.HandlerFunc

func ParseConfigurationType(l logrus.FieldLogger, next ConfigurationTypeHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var val string
		var ok bool
		if val, ok = mux.Vars(r)["type"]; !ok {
			l.Errorf("Unable to properly parse type from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(val)(w, r)
	}
}

type RegionHandler func(region string) http.HandlerFunc

func ParseRegion(l logrus.FieldLogger, next RegionHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var val string
		var ok bool
		if val, ok = mux.Vars(r)["region"]; !ok {
			l.Errorf("Unable to properly parse region from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(val)(w, r)
	}
}

type MajorVersionHandler func(majorVersion uint16) http.HandlerFunc

func ParseMajorVersion(l logrus.FieldLogger, next MajorVersionHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		majorVersion, err := strconv.Atoi(mux.Vars(r)["majorVersion"])
		if err != nil {
			l.WithError(err).Errorf("Unable to properly parse majorVersion from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(uint16(majorVersion))(w, r)
	}
}

type MinorVersionHandler func(minorVersion uint16) http.HandlerFunc

func ParseMinorVersion(l logrus.FieldLogger, next MinorVersionHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		minorVersion, err := strconv.Atoi(mux.Vars(r)["minorVersion"])
		if err != nil {
			l.WithError(err).Errorf("Unable to properly parse minorVersion from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(uint16(minorVersion))(w, r)
	}
}

type TenantIdHandler func(tenantId uuid.UUID) http.HandlerFunc

func ParseTenantId(l logrus.FieldLogger, next TenantIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantId, err := uuid.Parse(mux.Vars(r)["tenantId"])
		if err != nil {
			l.WithError(err).Errorf("Unable to properly parse tenantId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(tenantId)(w, r)
	}
}

type TemplateIdHandler func(templateId uuid.UUID) http.HandlerFunc

func ParseTemplateId(l logrus.FieldLogger, next TemplateIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		templateId, err := uuid.Parse(mux.Vars(r)["templateId"])
		if err != nil {
			l.WithError(err).Errorf("Unable to properly parse templateId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(templateId)(w, r)
	}
}

type ServiceIdHandler func(serviceId uuid.UUID) http.HandlerFunc

func ParseServiceId(l logrus.FieldLogger, next ServiceIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serviceId, err := uuid.Parse(mux.Vars(r)["serviceId"])
		if err != nil {
			l.WithError(err).Errorf("Unable to properly parse serviceId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(serviceId)(w, r)
	}
}
