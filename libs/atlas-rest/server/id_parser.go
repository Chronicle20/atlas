package server

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type IntegerId interface {
	~uint32 | ~int32 | ~int8 | ~uint8 | ~uint16
}

func ParseIntId[T IntegerId](l logrus.FieldLogger, varName string, next func(T) http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		value, err := strconv.Atoi(mux.Vars(r)[varName])
		if err != nil {
			l.WithError(err).Errorf("Error parsing %s as integer", varName)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(T(value))(w, r)
	}
}

func ParseUUIDId(l logrus.FieldLogger, varName string, next func(uuid.UUID) http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(mux.Vars(r)[varName])
		if err != nil {
			l.WithError(err).Errorf("Error parsing %s as uuid", varName)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(id)(w, r)
	}
}

func ParseStringId(l logrus.FieldLogger, varName string, next func(string) http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		value, ok := mux.Vars(r)[varName]
		if !ok {
			l.Errorf("%s not provided in path.", varName)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(value)(w, r)
	}
}
