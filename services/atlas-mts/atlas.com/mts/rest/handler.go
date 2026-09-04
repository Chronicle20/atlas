package rest

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
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
	return func(w http.ResponseWriter, r *http.Request) {
		worldIdStr, ok := mux.Vars(r)["worldId"]
		if !ok {
			l.Errorf("Unable to properly parse worldId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		worldId, err := strconv.ParseUint(worldIdStr, 10, 8)
		if err != nil {
			l.WithError(err).Errorf("Unable to properly parse worldId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(world.Id(byte(worldId)))(w, r)
	}
}

// ParseCharacterId parses the {characterId} path var into a uint32.
func ParseCharacterId(l logrus.FieldLogger, next func(characterId uint32) http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		characterIdStr, ok := mux.Vars(r)["characterId"]
		if !ok {
			l.Errorf("Unable to properly parse characterId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		characterId, err := strconv.ParseUint(characterIdStr, 10, 32)
		if err != nil {
			l.WithError(err).Errorf("Unable to properly parse characterId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(uint32(characterId))(w, r)
	}
}

// ParseAccountId parses the {accountId} path var into a uint32.
func ParseAccountId(l logrus.FieldLogger, next func(accountId uint32) http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		accountIdStr, ok := mux.Vars(r)["accountId"]
		if !ok {
			l.Errorf("Unable to properly parse accountId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		accountId, err := strconv.ParseUint(accountIdStr, 10, 32)
		if err != nil {
			l.WithError(err).Errorf("Unable to properly parse accountId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(uint32(accountId))(w, r)
	}
}

// ParseListingId parses the {listingId} path var (a UUID string).
func ParseListingId(l logrus.FieldLogger, next func(listingId string) http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		listingId, ok := mux.Vars(r)["listingId"]
		if !ok || listingId == "" {
			l.Errorf("Unable to properly parse listingId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(listingId)(w, r)
	}
}

// ParseHoldingId parses the {holdingId} path var (a UUID string).
func ParseHoldingId(l logrus.FieldLogger, next func(holdingId string) http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		holdingId, ok := mux.Vars(r)["holdingId"]
		if !ok || holdingId == "" {
			l.Errorf("Unable to properly parse holdingId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(holdingId)(w, r)
	}
}
