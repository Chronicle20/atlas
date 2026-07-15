package monster

import (
	"atlas-data/rest"
	"atlas-data/searchindex"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
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
				handleSearchMonsters(db)(d, c)(searchQuery)(w, r)
				return
			}

			page, err := paginate.ParseParams(query, paginate.DefaultPageSize, paginate.MaxPageSize)
			if err != nil {
				server.WriteBadRequest(d.Logger(), w, err.Error())
				return
			}

			s := NewStorage(d.Logger(), db)
			paged, err := s.AllPagedProvider(d.Context())(page)()
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to retrieve monsters.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(paged.Items, paginate.EnvelopeFor(paged), r)
		}
	}
}

func handleSearchMonsters(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) func(q string) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) func(q string) http.HandlerFunc {
		return func(q string) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				if q == "" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				if len(q) > searchindex.MaxQueryLen {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				query := r.URL.Query()
				page, err := paginate.ParseParams(query, searchindex.MaxLimit, searchindex.MaxLimit)
				if err != nil {
					server.WriteBadRequest(d.Logger(), w, err.Error())
					return
				}

				spec := searchindex.QuerySpec[SearchIndexEntity]{
					EntityIdColumn: "monster_id",
					NameColumns:    []string{"name"},
					Order:          "name ASC, monster_id ASC",
				}
				tenantId, err := searchindex.ResolveTenantId(db, d.Context(), spec)
				if err != nil {
					d.Logger().WithError(err).Errorf("Monster tenant resolve failed.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				offset := (page.Number - 1) * page.Size
				start := time.Now()
				rows, err := searchindex.Search(db, d.Context(), tenantId, q, offset, page.Size, spec)
				var total int
				if err == nil {
					total, err = searchindex.Count(db, d.Context(), tenantId, q, spec)
				}
				elapsedMs := time.Since(start).Milliseconds()
				if err != nil {
					d.Logger().WithError(err).Errorf("Monster search failed.")
					server.WriteErrorResponse(d.Logger())(w)(err)
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

				env := paginate.Envelope{Total: total, PageNumber: page.Number, PageSize: page.Size}
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalPaginatedResponse[[]SearchResultRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rms, env, r)
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
				query := r.URL.Query()
				page, err := paginate.ParseParams(query, paginate.DefaultPageSize, paginate.MaxPageSize)
				if err != nil {
					server.WriteBadRequest(d.Logger(), w, err.Error())
					return
				}

				t, terr := tenant.FromContext(d.Context())()
				if terr != nil {
					d.Logger().WithError(terr).Errorf("Unable to resolve tenant for monster-maps request.")
					server.WriteErrorResponse(d.Logger())(w)(terr)
					return
				}

				var rows []SpawnIndexEntity
				if err := db.WithContext(d.Context()).
					Where("tenant_id = ? AND monster_id = ?", t.Id(), monsterId).
					Order("spawn_count DESC, name ASC").
					Find(&rows).Error; err != nil {
					d.Logger().WithError(err).Errorf("Unable to query monster spawn index for monster %d.", monsterId)
					server.WriteErrorResponse(d.Logger())(w)(err)
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

				paged := paginate.Slice(rms, page)
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalPaginatedResponse[[]MonsterSpawnMapRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(paged.Items, paginate.EnvelopeFor(paged), r)
			}
		})
	}
}

func handleGetMonsterLoseItemsRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseMonsterId(d.Logger(), func(monsterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				query := r.URL.Query()
				page, err := paginate.ParseParams(query, paginate.DefaultPageSize, paginate.MaxPageSize)
				if err != nil {
					server.WriteBadRequest(d.Logger(), w, err.Error())
					return
				}

				s := NewStorage(d.Logger(), db)
				res, err := s.GetById(d.Context())(strconv.Itoa(int(monsterId)))
				if err != nil {
					d.Logger().WithError(err).Debugf("Unable to locate monster %d.", monsterId)
					w.WriteHeader(http.StatusNotFound)
					return
				}

				paged := paginate.Slice(res.LoseItems, page)
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalPaginatedResponse[[]loseItem](d.Logger())(w)(c.ServerInformation())(queryParams)(paged.Items, paginate.EnvelopeFor(paged), r)
			}
		})
	}
}
