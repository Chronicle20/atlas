package item

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

func InitStringResource(db *gorm.DB) func(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(si jsonapi.ServerInformation) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(si)

			r := router.PathPrefix("/data/item-strings").Subrouter()
			r.HandleFunc("", registerGet("get_item_strings", handleGetItemStringsRequest(db))).Methods(http.MethodGet)
			r.HandleFunc("/{itemId}", registerGet("get_item_string", handleGetItemStringRequest(db))).Methods(http.MethodGet)
		}
	}
}

type StringSearchResultRestModel struct {
	Id          string `json:"-"`
	Name        string `json:"name"`
	Compartment string `json:"compartment"`
	Subcategory string `json:"subcategory"`
}

func (r StringSearchResultRestModel) GetName() string { return "item-strings" }
func (r StringSearchResultRestModel) GetID() string   { return r.Id }

func (r *StringSearchResultRestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

func handleGetItemStringsRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query()
			searchQuery := strings.TrimSpace(query.Get("search"))
			if _, hasSearch := query["search"]; hasSearch {
				handleSearchItemStrings(db)(d, c)(searchQuery, query.Get("limit"))(w, r)
				return
			}

			s := NewStringStorage(d.Logger(), db)
			res, err := s.GetAll(d.Context())
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to retrieve item strings.")
				w.WriteHeader(http.StatusNotFound)
				return
			}

			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[[]StringRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	}
}

func handleSearchItemStrings(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) func(q, limitRaw string) http.HandlerFunc {
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

				start := time.Now()
				rows, err := searchindex.Search(db, d.Context(), q, limit, searchindex.QuerySpec[StringSearchIndexEntity]{
					EntityIdColumn: "item_id",
					NameColumns:    []string{"name"},
					Order:          "name ASC, item_id ASC",
					IdOf:           func(e StringSearchIndexEntity) uint64 { return uint64(e.ItemId) },
				})
				elapsedMs := time.Since(start).Milliseconds()
				if err != nil {
					d.Logger().WithError(err).Errorf("Item-string search failed.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				if t, terr := tenant.FromContext(d.Context())(); terr == nil {
					d.Logger().WithFields(logrus.Fields{
						"tenant_id":  t.Id().String(),
						"query_len":  len(q),
						"result_ct":  len(rows),
						"elapsed_ms": elapsedMs,
					}).Debugf("Item-string search served.")
				}

				rms := make([]StringSearchResultRestModel, 0, len(rows))
				for _, row := range rows {
					rms = append(rms, StringSearchResultRestModel{Id: strconv.Itoa(int(row.ItemId)), Name: row.Name})
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[[]StringSearchResultRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rms)
			}
		}
	}
}

func handleGetItemStringRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseItemId(d.Logger(), func(itemId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				s := NewStringStorage(d.Logger(), db)
				res, err := s.GetById(d.Context())(strconv.Itoa(int(itemId)))
				if err != nil {
					d.Logger().WithError(err).Debugf("Unable to locate item string for %d.", itemId)
					w.WriteHeader(http.StatusNotFound)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[StringRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
			}
		})
	}
}
