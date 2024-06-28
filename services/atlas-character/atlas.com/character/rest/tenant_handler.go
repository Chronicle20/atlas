package rest

import (
	"atlas-character/tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"net/http"
	"strconv"
)

const (
	ID           = "TENANT_ID"
	Region       = "REGION"
	MajorVersion = "MAJOR_VERSION"
	MinorVersion = "MINOR_VERSION"
)

type TenantHandler func(tenant tenant.Model) http.HandlerFunc

func ParseTenant(l logrus.FieldLogger, next TenantHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.Header.Get(ID)
		if idStr == "" {
			l.Errorf("%s is not supplied.", ID)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		id, err := uuid.Parse(idStr)
		if err != nil {
			l.Errorf("%s is not supplied.", ID)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		region := r.Header.Get(Region)
		if region == "" {
			l.Errorf("%s is not supplied.", Region)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		majorVersion := r.Header.Get(MajorVersion)
		if majorVersion == "" {
			l.Errorf("%s is not supplied.", MajorVersion)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		majorVersionVal, err := strconv.Atoi(majorVersion)
		if err != nil {
			l.Errorf("%s is not supplied.", MajorVersion)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		minorVersion := r.Header.Get(MinorVersion)
		if minorVersion == "" {
			l.Errorf("%s is not supplied.", MinorVersion)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		minorVersionVal, err := strconv.Atoi(minorVersion)
		if err != nil {
			l.Errorf("%s is not supplied.", MinorVersion)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		next(tenant.New(id, region, uint16(majorVersionVal), uint16(minorVersionVal)))(w, r)
	}
}
