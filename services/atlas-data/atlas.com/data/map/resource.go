package _map

import (
	"atlas-data/map/monster"
	"atlas-data/map/npc"
	"atlas-data/map/portal"
	"atlas-data/map/reactor"
	"atlas-data/point"
	"atlas-data/rest"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"net/http"
	"strconv"
)

func InitResource(db *gorm.DB) func(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(si jsonapi.ServerInformation) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(si)

			r := router.PathPrefix("/data/maps").Subrouter()
			r.HandleFunc("", registerGet("get_maps", handleGetMapsRequest(db))).Methods(http.MethodGet)
			r.HandleFunc("/{mapId}", registerGet("get_map", handleGetMapRequest(db))).Methods(http.MethodGet)
			r.HandleFunc("/{mapId}/portals", registerGet("get_map_portals_by_name", handleGetMapPortalsByNameRequest(db))).Queries("name", "{name}").Methods(http.MethodGet)
			r.HandleFunc("/{mapId}/portals", registerGet("get_map_portals", handleGetMapPortalsRequest(db))).Methods(http.MethodGet)
			r.HandleFunc("/{mapId}/portals/{portalId}", registerGet("get_map_portal", handleGetMapPortalRequest(db))).Methods(http.MethodGet)
			r.HandleFunc("/{mapId}/reactors", registerGet("get_map_reactors", handleGetMapReactorsRequest(db))).Methods(http.MethodGet)
			r.HandleFunc("/{mapId}/npcs", registerGet("get_map_npcs_by_object_id", handleGetMapNPCsByObjectIdRequest(db))).Queries("objectId", "{objectId}").Methods(http.MethodGet)
			r.HandleFunc("/{mapId}/npcs", registerGet("get_map_npcs", handleGetMapNPCsRequest(db))).Methods(http.MethodGet)
			r.HandleFunc("/{mapId}/npcs/{npcId}", registerGet("get_map_npc", handleGetMapNPCRequest(db))).Methods(http.MethodGet)
			r.HandleFunc("/{mapId}/monsters", registerGet("get_map_monsters", handleGetMapMonstersRequest(db))).Methods(http.MethodGet)
			r.HandleFunc("/{mapId}/drops/position", rest.RegisterInputHandler[DropPositionRestModel](l)(si)("get_map_drop_position", handleGetMapDropPositionRequest(db))).Methods(http.MethodPost)
			r.HandleFunc("/{mapId}/footholds/below", rest.RegisterInputHandler[PositionRestModel](l)(si)("get_map_foothold_below", handleGetMapFootholdBelowRequest(db))).Methods(http.MethodPost)
		}
	}
}

func handleGetMapsRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			s := NewStorage(d.Logger(), db)
			m, err := s.GetAll(d.Context())
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			res, err := model.SliceMap(Transform)(model.FixedProvider(m))()()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	}
}

func handleGetMapRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseMapId(d.Logger(), func(mapId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				s := NewStorage(d.Logger(), db)
				m, err := s.GetById(d.Context())(mapId)
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
}

func handleGetMapPortalsByNameRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseMapId(d.Logger(), func(mapId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				vars := mux.Vars(r)
				portalName := vars["name"]

				s := NewStorage(d.Logger(), db)
				ps, err := GetPortalsByName(s)(d.Context())(mapId, portalName)
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
}

func handleGetMapPortalsRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseMapId(d.Logger(), func(mapId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				s := NewStorage(d.Logger(), db)
				ps, err := GetPortals(s)(d.Context())(mapId)
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
}

func handleGetMapPortalRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseMapId(d.Logger(), func(mapId uint32) http.HandlerFunc {
			return rest.ParsePortalId(d.Logger(), func(portalId uint32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					s := NewStorage(d.Logger(), db)
					p, err := GetPortalById(s)(d.Context())(mapId, portalId)
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
}

func handleGetMapReactorsRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseMapId(d.Logger(), func(mapId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				s := NewStorage(d.Logger(), db)
				rs, err := GetReactors(s)(d.Context())(mapId)
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
}

func handleGetMapNPCsByObjectIdRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseMapId(d.Logger(), func(mapId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				vars := mux.Vars(r)
				objectId, err := strconv.Atoi(vars["objectId"])
				if err != nil {
					d.Logger().WithError(err).Errorf("Error parsing objectId as uint32")
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				s := NewStorage(d.Logger(), db)
				ns, err := GetNpcsByObjectId(s)(d.Context())(mapId, uint32(objectId))
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
}

func handleGetMapNPCsRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseMapId(d.Logger(), func(mapId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				s := NewStorage(d.Logger(), db)
				ns, err := GetNpcs(s)(d.Context())(mapId)
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
}

func handleGetMapNPCRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseMapId(d.Logger(), func(mapId uint32) http.HandlerFunc {
			return rest.ParseNPC(d.Logger(), func(npcId uint32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					s := NewStorage(d.Logger(), db)
					ns, err := GetNpc(s)(d.Context())(mapId, npcId)
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
}

func handleGetMapMonstersRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseMapId(d.Logger(), func(mapId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				s := NewStorage(d.Logger(), db)
				ms, err := GetMonsters(s)(d.Context())(mapId)
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
}

func handleGetMapDropPositionRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext, input DropPositionRestModel) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, input DropPositionRestModel) http.HandlerFunc {
		return rest.ParseMapId(d.Logger(), func(mapId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				s := NewStorage(d.Logger(), db)
				p := calcDropPos(s)(d.Context())(mapId, &point.Model{X: input.InitialX, Y: input.InitialY}, &point.Model{X: input.FallbackX, Y: input.FallbackY})
				res, err := model.Map(point.Transform)(model.FixedProvider(*p))()
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				server.Marshal[point.RestModel](d.Logger())(w)(c.ServerInformation())(res)
			}
		})
	}
}

type DropPositionRestModel struct {
	Id        uint32 `json:"-"`
	InitialX  int16  `json:"initialX"`
	InitialY  int16  `json:"initialY"`
	FallbackX int16  `json:"fallbackX"`
	FallbackY int16  `json:"fallbackY"`
}

func (r DropPositionRestModel) GetName() string {
	return "positions"
}

func (r DropPositionRestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *DropPositionRestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func handleGetMapFootholdBelowRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext, i PositionRestModel) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, i PositionRestModel) http.HandlerFunc {
		return rest.ParseMapId(d.Logger(), func(mapId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				s := NewStorage(d.Logger(), db)
				m, err := s.GetById(d.Context())(mapId)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				p := &point.Model{X: i.X, Y: i.Y}
				fh := m.FootholdTree.findBelow(p)
				if fh == nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				rm := FootholdRestModel{
					Id: fh.Id,
					X1: fh.First.X,
					Y1: fh.First.Y,
					X2: fh.Second.X,
					Y2: fh.Second.Y,
				}
				server.Marshal[FootholdRestModel](d.Logger())(w)(c.ServerInformation())(rm)
			}
		})
	}
}

type PositionRestModel struct {
	Id uint32 `json:"-"`
	X  int16  `json:"x"`
	Y  int16  `json:"y"`
}

func (r PositionRestModel) GetName() string {
	return "positions"
}

func (r PositionRestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *PositionRestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

type FootholdRestModel struct {
	Id uint32 `json:"-"`
	X1 int16  `json:"x1"`
	Y1 int16  `json:"y1"`
	X2 int16  `json:"x2"`
	Y2 int16  `json:"y2"`
}

func (r FootholdRestModel) GetName() string {
	return "footholds"
}

func (r FootholdRestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *FootholdRestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}
