package tenant

import (
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

type Handler func(tenant Model) http.HandlerFunc

func ParseTenant(l logrus.FieldLogger, next Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(ID)
		if id == "" {
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

		next(Model{
			id:           uuid.MustParse(id),
			region:       region,
			majorVersion: uint16(majorVersionVal),
			minorVersion: uint16(minorVersionVal),
		})(w, r)
	}
}
