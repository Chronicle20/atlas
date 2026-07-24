package broadcast

import (
	"atlas-world/rest"
	"errors"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

const getBroadcastQueue = "get_broadcast_queue"

// InitResource mounts GET /worlds/{worldId}/broadcast-queues/{family},
// returning the current QueueModel for that (world, family) as a
// broadcast-queues JSON:API resource.
func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		registerGet := rest.RegisterHandler(l)(si)

		r := router.PathPrefix("/worlds/{worldId}/broadcast-queues").Subrouter()
		r.HandleFunc("/{family}", registerGet(getBroadcastQueue, handleGetBroadcastQueue)).Methods(http.MethodGet)
	}
}

func handleGetBroadcastQueue(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseWorldId(d.Logger(), func(worldId world.Id) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			family := mux.Vars(r)["family"]
			if family != FamilyTV && family != FamilyAvatar {
				server.WriteBadRequest(d.Logger(), w, "family must be TV or AVATAR")
				return
			}

			q, err := NewProcessor(d.Logger(), d.Context()).GetQueue(worldId, family)
			if err != nil {
				if !errors.Is(err, atlas.ErrNotFound) {
					d.Logger().WithError(err).Errorf("Unable to get broadcast queue for world [%d] family [%s].", worldId, family)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				// No queue created yet for this (world, family) - treat as
				// idle rather than surfacing the storage-layer miss.
				q = QueueModel{}
			}

			rm, err := Transform(family, q, time.Now())
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating broadcast queue REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
		}
	})
}
