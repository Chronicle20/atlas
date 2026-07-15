package location

import (
	"atlas-maps/data/map/info"
	"atlas-maps/rest"
	"context"
	"errors"
	"net/http"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// WarpProvider constructs the warp processor used by the PATCH handler. It is
// injected from main.go so the location package does not import the warp
// package (warp already imports location, which would be an import cycle).
type WarpProvider func(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) WarpProcessor

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB, wp WarpProvider) server.RouteInitializer {
	return func(db *gorm.DB, wp WarpProvider) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerHandler := rest.RegisterHandler(l)(si)

			r := router.PathPrefix("/characters").Subrouter()
			r.HandleFunc("/{characterId}/location", registerHandler("get_character_location", handleGetCharacterLocation(db))).Methods(http.MethodGet)
			r.HandleFunc("/{characterId}/location",
				rest.RegisterInputHandler[RestModel](l)(si)("change_character_location", handleChangeCharacterLocation(db, wp)),
			).Methods(http.MethodPatch)
		}
	}
}

// WarpProcessor is the narrow slice of warp.Processor the helper needs.
type WarpProcessor interface {
	ChangeMap(transactionId uuid.UUID, characterId uint32, worldId world.Id, dest field.Model, portalId uint32, useTargetPosition bool, targetX int16, targetY int16) error
}

func handleChangeCharacterLocation(db *gorm.DB, wp WarpProvider) rest.InputHandler[RestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				lp := NewProcessor(d.Logger(), d.Context(), db)
				ip := info.NewProcessor(d.Logger(), d.Context())
				status, err := changeCharacterLocation(d.Logger(), lp, ip, wp(d.Logger(), d.Context(), db), characterId, input.MapId)
				if status == http.StatusInternalServerError {
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				w.WriteHeader(status)
			}
		})
	}
}

// changeCharacterLocation is the unit-testable core of the write handler. It
// returns the HTTP status to write. channelId/instance from the body are
// ignored — this is a map-only warp; destination channel is the stored channel,
// instance is uuid.Nil (non-instanced), spawn portal 0.
func changeCharacterLocation(l logrus.FieldLogger, lp Processor, ip info.Processor, wp WarpProcessor, characterId uint32, targetMapId _map.Id) (int, error) {
	cur, err := lp.GetById(characterId)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		l.Warnf("change_character_location: no location row for character [%d]; rejecting 404.", characterId)
		return http.StatusNotFound, nil
	}
	if err != nil {
		l.WithError(err).Errorf("change_character_location: loading location for character [%d].", characterId)
		return http.StatusInternalServerError, err
	}

	if _, err := ip.GetById(targetMapId); err != nil {
		if errors.Is(err, requests.ErrNotFound) {
			l.WithError(err).Warnf("change_character_location: target map [%d] does not exist; rejecting 400.", targetMapId)
			return http.StatusBadRequest, nil
		}
		l.WithError(err).Errorf("change_character_location: map-existence check failed for [%d] (infrastructure).", targetMapId)
		return http.StatusInternalServerError, err
	}

	dest := field.NewBuilder(cur.WorldId(), cur.ChannelId(), targetMapId).SetInstance(uuid.Nil).Build()
	if err := wp.ChangeMap(uuid.New(), characterId, cur.WorldId(), dest, 0, false, 0, 0); err != nil {
		l.WithError(err).Errorf("change_character_location: warp failed for character [%d].", characterId)
		return http.StatusInternalServerError, err
	}

	l.WithFields(logrus.Fields{"character_id": characterId, "map_id": targetMapId}).
		Infof("change_character_location: warped character [%d] to map [%d].", characterId, targetMapId)
	return http.StatusNoContent, nil
}

func handleGetCharacterLocation(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				m, err := NewProcessor(d.Logger(), d.Context(), db).GetById(characterId)
				if errors.Is(err, gorm.ErrRecordNotFound) {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to load location for character [%d].", characterId)
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				rm, err := model.Map(Transform)(model.FixedProvider(m))()
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
			}
		})
	}
}
