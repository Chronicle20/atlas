package monster

import (
	"atlas-data/rest"
	"atlas-data/searchindex"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitResource(db *gorm.DB) func(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(si jsonapi.ServerInformation) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(si)

			r := router.PathPrefix("/data/monsters").Subrouter()
			r.HandleFunc("", registerGet("get_monsters", handleGetMonstersRequest(db))).Methods(http.MethodGet)
			r.HandleFunc("/{monsterId}", registerGet("get_monster", handleGetMonsterRequest(db))).Methods(http.MethodGet)
			r.HandleFunc("/{monsterId}/loseItems", registerGet("get_monster_lose_items", handleGetMonsterLoseItemsRequest(db))).Methods(http.MethodGet)
			r.HandleFunc("/{monsterId}/maps", registerGet("get_monster_maps", handleGetMonsterMapsRequest(db))).Methods(http.MethodGet)
		}
	}
}

type SearchResultRestModel struct {
	Id   uint32 `json:"-"`
	Name string `json:"name"`
}

func (r SearchResultRestModel) GetName() string { return "monsters" }
func (r SearchResultRestModel) GetID() string   { return strconv.Itoa(int(r.Id)) }

func (r *SearchResultRestModel) SetID(idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func handleGetMonstersRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query()
			searchQuery := strings.TrimSpace(query.Get("search"))
			if _, hasSearch := query["search"]; hasSearch {
				handleSearchMonsters(db)(d, c)(searchQuery, query.Get("limit"))(w, r)
				return
			}

			s := NewStorage(d.Logger(), db)
			results, err := s.GetAll(d.Context())
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to retrieve monsters.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(results)
		}
	}
}

func handleSearchMonsters(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) func(q, limitRaw string) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) func(q, limitRaw string) http.HandlerFunc {
		return func(q, limitRaw string) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				if q == "" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				if len(q) > searchindex.MaxQueryLen {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				limit := searchindex.MaxLimit
				if limitRaw != "" {
					parsed, err := strconv.Atoi(limitRaw)
					if err != nil || parsed <= 0 {
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					if parsed > searchindex.MaxLimit {
						parsed = searchindex.MaxLimit
					}
					limit = parsed
				}

				spec := searchindex.QuerySpec[SearchIndexEntity]{
					EntityIdColumn: "monster_id",
					NameColumns:    []string{"name"},
					Order:          "name ASC, monster_id ASC",
				}
				tenantId, err := searchindex.ResolveTenantId(db, d.Context(), spec)
				if err != nil {
					d.Logger().WithError(err).Errorf("Monster tenant resolve failed.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				start := time.Now()
				rows, err := searchindex.Search(db, d.Context(), tenantId, q, 0, limit, spec)
				elapsedMs := time.Since(start).Milliseconds()
				if err != nil {
					d.Logger().WithError(err).Errorf("Monster search failed.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				if t, terr := tenant.FromContext(d.Context())(); terr == nil {
					d.Logger().WithFields(logrus.Fields{
						"tenant_id":  t.Id().String(),
						"query_len":  len(q),
						"result_ct":  len(rows),
						"elapsed_ms": elapsedMs,
					}).Debugf("Monster search served.")
				}

				rms := make([]SearchResultRestModel, 0, len(rows))
				for _, row := range rows {
					rms = append(rms, SearchResultRestModel{Id: row.MonsterId, Name: row.Name})
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[[]SearchResultRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rms)
			}
		}
	}
}

func handleGetMonsterRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseMonsterId(d.Logger(), func(monsterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				s := NewStorage(d.Logger(), db)
				res, err := s.GetById(d.Context())(strconv.Itoa(int(monsterId)))
				if err != nil {
					d.Logger().WithError(err).Debugf("Unable to locate monster %d.", monsterId)
					w.WriteHeader(http.StatusNotFound)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
			}
		})
	}
}

type MonsterSpawnMapRestModel struct {
	MapId      uint32 `json:"-"`
	Name       string `json:"name"`
	StreetName string `json:"streetName"`
	SpawnCount uint32 `json:"spawnCount"`
}

func (r MonsterSpawnMapRestModel) GetName() string { return "monster-spawn-maps" }
func (r MonsterSpawnMapRestModel) GetID() string   { return strconv.Itoa(int(r.MapId)) }

func (r *MonsterSpawnMapRestModel) SetID(idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}
	r.MapId = uint32(id)
	return nil
}

func handleGetMonsterMapsRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseMonsterId(d.Logger(), func(monsterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				t, terr := tenant.FromContext(d.Context())()
				if terr != nil {
					d.Logger().WithError(terr).Errorf("Unable to resolve tenant for monster-maps request.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				var rows []SpawnIndexEntity
				if err := db.WithContext(d.Context()).
					Where("tenant_id = ? AND monster_id = ?", t.Id(), monsterId).
					Order("spawn_count DESC, name ASC").
					Find(&rows).Error; err != nil {
					d.Logger().WithError(err).Errorf("Unable to query monster spawn index for monster %d.", monsterId)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				rms := make([]MonsterSpawnMapRestModel, 0, len(rows))
				for _, row := range rows {
					rms = append(rms, MonsterSpawnMapRestModel{
						MapId:      row.MapId,
						Name:       row.Name,
						StreetName: row.StreetName,
						SpawnCount: row.SpawnCount,
					})
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[[]MonsterSpawnMapRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rms)
			}
		})
	}
}

func handleGetMonsterLoseItemsRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseMonsterId(d.Logger(), func(monsterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				s := NewStorage(d.Logger(), db)
				res, err := s.GetById(d.Context())(strconv.Itoa(int(monsterId)))
				if err != nil {
					d.Logger().WithError(err).Debugf("Unable to locate monster %d.", monsterId)
					w.WriteHeader(http.StatusNotFound)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[[]loseItem](d.Logger())(w)(c.ServerInformation())(queryParams)(res.LoseItems)
			}
		})
	}
}
