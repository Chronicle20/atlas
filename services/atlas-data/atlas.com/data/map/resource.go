package _map

import (
	"atlas-data/map/monster"
	"atlas-data/map/npc"
	point2 "atlas-data/map/point"
	"atlas-data/map/portal"
	"atlas-data/map/reactor"
	"atlas-data/point"
	"atlas-data/rest"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/gorilla/mux"
	"github.com/manyminds/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"net/http"
	"strconv"
)

const (
	getMap               = "get_map"
	getMapPortalsByName  = "get_map_portals_by_name"
	getMapPortals        = "get_map_portals"
	getMapPortal         = "get_map_portal"
	getMapReactors       = "get_map_reactors"
	getMapNPCsByObjectId = "get_map_npcs_by_object_id"
	getMapNPCs           = "get_map_npcs"
	getMapNPC            = "get_map_npc"
	getMapMonsters       = "get_map_monsters"
	getMapDropPosition   = "get_map_drop_position"
)

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		registerGet := rest.RegisterHandler(l)(si)
		registerInput := rest.RegisterInputHandler[DropPositionRestModel](l)(si)

		r := router.PathPrefix("/maps").Subrouter()
		r.HandleFunc("/{mapId}", registerGet(getMap, handleGetMapRequest)).Methods(http.MethodGet)
		r.HandleFunc("/{mapId}/portals", registerGet(getMapPortalsByName, handleGetMapPortalsByNameRequest)).Queries("name", "{name}").Methods(http.MethodGet)
		r.HandleFunc("/{mapId}/portals", registerGet(getMapPortals, handleGetMapPortalsRequest)).Methods(http.MethodGet)
		r.HandleFunc("/{mapId}/portals/{portalId}", registerGet(getMapPortal, handleGetMapPortalRequest)).Methods(http.MethodGet)
		r.HandleFunc("/{mapId}/reactors", registerGet(getMapReactors, handleGetMapReactorsRequest)).Methods(http.MethodGet)
		r.HandleFunc("/{mapId}/npcs", registerGet(getMapNPCsByObjectId, handleGetMapNPCsByObjectIdRequest)).Queries("objectId", "{objectId}").Methods(http.MethodGet)
		r.HandleFunc("/{mapId}/npcs", registerGet(getMapNPCs, handleGetMapNPCsRequest)).Methods(http.MethodGet)
		r.HandleFunc("/{mapId}/npcs/{npcId}", registerGet(getMapNPC, handleGetMapNPCRequest)).Methods(http.MethodGet)
		r.HandleFunc("/{mapId}/monsters", registerGet(getMapMonsters, handleGetMapMonstersRequest)).Methods(http.MethodGet)
		r.HandleFunc("/{mapId}/dropPosition", registerInput(getMapDropPosition, handleGetMapDropPositionRequest)).Methods(http.MethodPost)
	}
}

func handleGetMapRequest(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseMapId(d.Logger(), func(mapId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			m, err := GetById(d.Context())(mapId)
			if err != nil {
				d.Logger().WithError(err).Debugf("Unable to locate map %d.", mapId)
				w.WriteHeader(http.StatusNotFound)
				return
			}

			res, err := model.Map(Transform)(model.FixedProvider(m))()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			server.Marshal[RestModel](d.Logger())(w)(c.ServerInformation())(res)
		}
	})
}

func handleGetMapPortalsRequest(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseMapId(d.Logger(), func(mapId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ps, err := GetPortals(d.Context())(mapId)
			if err != nil {
				d.Logger().WithError(err).Debugf("Unable to locate map %d.", mapId)
				w.WriteHeader(http.StatusNotFound)
				return
			}

			res, err := model.SliceMap(portal.Transform)(model.FixedProvider(ps))(model.ParallelMap())()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			server.Marshal[[]portal.RestModel](d.Logger())(w)(c.ServerInformation())(res)
		}
	})
}

func handleGetMapPortalsByNameRequest(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseMapId(d.Logger(), func(mapId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)
			portalName := vars["name"]

			ps, err := GetPortalsByName(d.Context())(mapId, portalName)
			if err != nil {
				d.Logger().WithError(err).Debugf("Unable to locate map %d.", mapId)
				w.WriteHeader(http.StatusNotFound)
				return
			}

			res, err := model.SliceMap(portal.Transform)(model.FixedProvider(ps))(model.ParallelMap())()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			server.Marshal[[]portal.RestModel](d.Logger())(w)(c.ServerInformation())(res)
		}
	})
}

func handleGetMapPortalRequest(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseMapId(d.Logger(), func(mapId uint32) http.HandlerFunc {
		return rest.ParsePortalId(d.Logger(), func(portalId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				p, err := GetPortalById(d.Context())(mapId, portalId)
				if err != nil {
					d.Logger().WithError(err).Debugf("Unable to locate map %d.", mapId)
					w.WriteHeader(http.StatusNotFound)
					return
				}

				res, err := model.Map(portal.Transform)(model.FixedProvider(p))()
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				server.Marshal[portal.RestModel](d.Logger())(w)(c.ServerInformation())(res)
			}
		})
	})
}

func handleGetMapReactorsRequest(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseMapId(d.Logger(), func(mapId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			rs, err := GetReactors(d.Context())(mapId)
			if err != nil {
				d.Logger().WithError(err).Debugf("Unable to locate map %d.", mapId)
				w.WriteHeader(http.StatusNotFound)
				return
			}

			res, err := model.SliceMap(reactor.Transform)(model.FixedProvider(rs))(model.ParallelMap())()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			server.Marshal[[]reactor.RestModel](d.Logger())(w)(c.ServerInformation())(res)
		}
	})
}

func handleGetMapNPCsByObjectIdRequest(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseMapId(d.Logger(), func(mapId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)
			objectId, err := strconv.Atoi(vars["objectId"])
			if err != nil {
				d.Logger().WithError(err).Errorf("Error parsing objectId as uint32")
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			ns, err := GetNpcsByObjectId(d.Context())(mapId, uint32(objectId))
			res, err := model.SliceMap(npc.Transform)(model.FixedProvider(ns))(model.ParallelMap())()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			server.Marshal[[]npc.RestModel](d.Logger())(w)(c.ServerInformation())(res)
		}
	})
}

func handleGetMapNPCsRequest(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseMapId(d.Logger(), func(mapId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ns, err := GetNpcs(d.Context())(mapId)
			res, err := model.SliceMap(npc.Transform)(model.FixedProvider(ns))(model.ParallelMap())()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			server.Marshal[[]npc.RestModel](d.Logger())(w)(c.ServerInformation())(res)
		}
	})
}

func handleGetMapNPCRequest(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseMapId(d.Logger(), func(mapId uint32) http.HandlerFunc {
		return rest.ParseNPC(d.Logger(), func(npcId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				ns, err := GetNpc(d.Context())(mapId, npcId)
				res, err := model.Map(npc.Transform)(model.FixedProvider(ns))()
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				server.Marshal[npc.RestModel](d.Logger())(w)(c.ServerInformation())(res)
			}
		})
	})
}

func handleGetMapMonstersRequest(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseMapId(d.Logger(), func(mapId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ms, err := GetMonsters(d.Context())(mapId)
			res, err := model.SliceMap(monster.Transform)(model.FixedProvider(ms))(model.ParallelMap())()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			server.Marshal[[]monster.RestModel](d.Logger())(w)(c.ServerInformation())(res)
		}
	})
}

func handleGetMapDropPositionRequest(d *rest.HandlerDependency, c *rest.HandlerContext, input DropPositionRestModel) http.HandlerFunc {
	return rest.ParseMapId(d.Logger(), func(mapId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			p := calcDropPos(tenant.MustFromContext(d.Context()), mapId, point.NewModel(input.InitialX, input.InitialY), point.NewModel(input.FallbackX, input.FallbackY))
			res, err := model.Map(point2.Transform)(model.FixedProvider(*p))()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			server.Marshal[point2.RestModel](d.Logger())(w)(c.ServerInformation())(res)
			w.WriteHeader(http.StatusCreated)
		}
	})
}

type DropPositionRestModel struct {
	InitialX  int16 `json:"initialX"`
	InitialY  int16 `json:"initialY"`
	FallbackX int16 `json:"fallbackX"`
	FallbackY int16 `json:"fallbackY"`
}
