package npc

import (
	"atlas-data/rest"
	"atlas-data/searchindex"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"atlas-data/quest"

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

			r := router.PathPrefix("/data/npcs").Subrouter()
			r.HandleFunc("", registerGet("get_npcs", handleGetNpcsRequest(db))).Methods(http.MethodGet)
			r.HandleFunc("/{npcId}", registerGet("get_npc", handleGetNpcRequest(db))).Methods(http.MethodGet)
			r.HandleFunc("/{npcId}/maps", registerGet("get_npc_maps", handleGetNpcMapsRequest(db))).Methods(http.MethodGet)
			r.HandleFunc("/{npcId}/quests", registerGet("get_npc_quests", handleGetNpcQuestsRequest(db))).Methods(http.MethodGet)
		}
	}
}

type SearchResultRestModel struct {
	Id        uint32 `json:"-"`
	Name      string `json:"name"`
	Storebank bool   `json:"storebank"`
}

func (r SearchResultRestModel) GetName() string { return "npcs" }
func (r SearchResultRestModel) GetID() string   { return strconv.Itoa(int(r.Id)) }

func (r *SearchResultRestModel) SetID(idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func handleGetNpcsRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query()
			searchQuery := strings.TrimSpace(query.Get("search"))
			_, hasSearch := query["search"]
			storebankFilter := query.Get("filter[storebank]") == "true"

			if hasSearch || storebankFilter {
				handleSearchNpcs(db)(d, c)(searchQuery, hasSearch, storebankFilter)(w, r)
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
				d.Logger().WithError(err).Errorf("Unable to retrieve NPCs.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(paged.Items, paginate.EnvelopeFor(paged), r)
		}
	}
}

func handleSearchNpcs(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) func(q string, hasSearch, storebank bool) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) func(q string, hasSearch, storebank bool) http.HandlerFunc {
		return func(q string, hasSearch, storebank bool) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				if hasSearch && q == "" {
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
					EntityIdColumn: "npc_id",
					NameColumns:    []string{"name"},
					Order:          "name ASC, npc_id ASC",
				}
				if storebank {
					spec.ExtraPredicate = "storebank = ?"
					spec.ExtraArgs = []interface{}{true}
				}

				tenantId, err := searchindex.ResolveTenantId(db, d.Context(), spec)
				if err != nil {
					d.Logger().WithError(err).Errorf("NPC tenant resolve failed.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				offset := (page.Number - 1) * page.Size
				start := time.Now()
				var rows []SearchIndexEntity
				var total int
				if hasSearch {
					rows, err = searchindex.Search(db, d.Context(), tenantId, q, offset, page.Size, spec)
					if err == nil {
						total, err = searchindex.Count(db, d.Context(), tenantId, q, spec)
					}
				} else {
					rows, err = searchindex.SearchWithFilter(db, d.Context(), tenantId, offset, page.Size, spec)
					if err == nil {
						total, err = searchindex.CountWithFilter(db, d.Context(), tenantId, spec)
					}
				}
				elapsedMs := time.Since(start).Milliseconds()
				if err != nil {
					d.Logger().WithError(err).Errorf("NPC search failed.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				if t, terr := tenant.FromContext(d.Context())(); terr == nil {
					d.Logger().WithFields(logrus.Fields{
						"tenant_id":  t.Id().String(),
						"query_len":  len(q),
						"result_ct":  len(rows),
						"elapsed_ms": elapsedMs,
						"storebank":  storebank,
					}).Debugf("NPC search served.")
				}

				rms := make([]SearchResultRestModel, 0, len(rows))
				for _, row := range rows {
					rms = append(rms, SearchResultRestModel{Id: row.NpcId, Name: row.Name, Storebank: row.Storebank})
				}

				env := paginate.Envelope{Total: total, PageNumber: page.Number, PageSize: page.Size}
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalPaginatedResponse[[]SearchResultRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rms, env, r)
			}
		}
	}
}

func handleGetNpcMapsRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseNPC(d.Logger(), func(npcId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				query := r.URL.Query()
				page, err := paginate.ParseParams(query, paginate.DefaultPageSize, paginate.MaxPageSize)
				if err != nil {
					server.WriteBadRequest(d.Logger(), w, err.Error())
					return
				}

				t, err := tenant.FromContext(d.Context())()
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				start := time.Now()
				var rows []SpawnIndexEntity
				qerr := db.WithContext(d.Context()).
					Where("tenant_id = ? AND npc_id = ?", t.Id(), npcId).
					Order("spawn_count DESC, map_id ASC").
					Find(&rows).Error
				if qerr != nil {
					d.Logger().WithError(qerr).Errorf("Unable to retrieve NPC spawn maps for npcId=%d.", npcId)
					server.WriteErrorResponse(d.Logger())(w)(qerr)
					return
				}

				seen := make(map[uint32]struct{}, len(rows))
				rms := make([]NpcMapRestModel, 0, len(rows))
				for _, row := range rows {
					if _, dup := seen[row.MapId]; dup {
						continue
					}
					seen[row.MapId] = struct{}{}
					rms = append(rms, NpcMapRestModel{
						NpcId:      row.NpcId,
						MapId:      row.MapId,
						Name:       row.Name,
						StreetName: row.StreetName,
						SpawnCount: row.SpawnCount,
					})
				}
				elapsedMs := time.Since(start).Milliseconds()

				d.Logger().WithFields(logrus.Fields{
					"tenant_id":  t.Id().String(),
					"npc_id":     npcId,
					"result_ct":  len(rms),
					"elapsed_ms": elapsedMs,
				}).Debugf("NPC spawn maps served.")

				paged := paginate.Slice(rms, page)
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalPaginatedResponse[[]NpcMapRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(paged.Items, paginate.EnvelopeFor(paged), r)
			}
		})
	}
}

func handleGetNpcQuestsRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseNPC(d.Logger(), func(npcId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				query := r.URL.Query()
				page, err := paginate.ParseParams(query, paginate.DefaultPageSize, paginate.MaxPageSize)
				if err != nil {
					server.WriteBadRequest(d.Logger(), w, err.Error())
					return
				}

				t, err := tenant.FromContext(d.Context())()
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				start := time.Now()
				s := quest.NewStorage(d.Logger(), db)
				all, qerr := s.DrainAllProvider(d.Context())()
				if qerr != nil {
					d.Logger().WithError(qerr).Errorf("Unable to retrieve quests for npcId=%d.", npcId)
					server.WriteErrorResponse(d.Logger())(w)(qerr)
					return
				}

				filtered := make([]quest.RestModel, 0)
				for _, q := range all {
					if q.StartRequirements.NpcId == npcId ||
						q.EndRequirements.NpcId == npcId ||
						q.StartActions.NpcId == npcId ||
						q.EndActions.NpcId == npcId {
						filtered = append(filtered, q)
					}
				}
				sort.SliceStable(filtered, func(i, j int) bool {
					return filtered[i].Id < filtered[j].Id
				})
				elapsedMs := time.Since(start).Milliseconds()

				d.Logger().WithFields(logrus.Fields{
					"tenant_id":  t.Id().String(),
					"npc_id":     npcId,
					"result_ct":  len(filtered),
					"elapsed_ms": elapsedMs,
				}).Debugf("NPC quests served.")

				paged := paginate.Slice(filtered, page)
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalPaginatedResponse[[]quest.RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(paged.Items, paginate.EnvelopeFor(paged), r)
			}
		})
	}
}

func handleGetNpcRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseNPC(d.Logger(), func(npcId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				s := NewStorage(d.Logger(), db)
				res, err := s.GetById(d.Context())(strconv.Itoa(int(npcId)))
				if err != nil {
					d.Logger().WithError(err).Debugf("Unable to locate NPC %d.", npcId)
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
