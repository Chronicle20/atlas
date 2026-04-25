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
			searchKey, hasSearch := query["search"]
			searchQuery := ""
			if hasSearch {
				searchQuery = strings.TrimSpace(searchKey[0])
				if searchQuery == "" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				if len(searchQuery) > searchindex.MaxQueryLen {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
			}

			limit := searchindex.MaxLimit
			if raw := query.Get("limit"); raw != "" {
				parsed, err := strconv.Atoi(raw)
				if err != nil || parsed <= 0 {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				if parsed > searchindex.MaxLimit {
					parsed = searchindex.MaxLimit
				}
				limit = parsed
			}

			fspec, errCode := parseFilters(query)
			if errCode != 0 {
				w.WriteHeader(errCode)
				return
			}

			hasFilter := fspec.Compartment != nil || fspec.Subcategory != "" || fspec.Class != ""

			spec := searchindex.QuerySpec[StringSearchIndexEntity]{
				EntityIdColumn: "item_id",
				NameColumns:    []string{"name"},
				Order:          "name ASC, item_id ASC",
				IdOf:           func(e StringSearchIndexEntity) uint64 { return uint64(e.ItemId) },
			}

			predicates, args := buildPredicates(fspec, !hasSearch && !hasFilter)
			if len(predicates) > 0 {
				spec.ExtraPredicate = strings.Join(predicates, " AND ")
				spec.ExtraArgs = args
			}

			start := time.Now()
			var rows []StringSearchIndexEntity
			var err error
			if hasSearch {
				rows, err = searchindex.Search(db, d.Context(), searchQuery, limit, spec)
			} else {
				rows, err = searchindex.SearchWithFilter(db, d.Context(), limit, spec)
			}
			elapsedMs := time.Since(start).Milliseconds()
			if err != nil {
				d.Logger().WithError(err).Errorf("Item-string search failed.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			if t, terr := tenant.FromContext(d.Context())(); terr == nil {
				d.Logger().WithFields(logrus.Fields{
					"tenant_id":    t.Id().String(),
					"query_len":    len(searchQuery),
					"result_ct":    len(rows),
					"elapsed_ms":   elapsedMs,
					"compartment":  compartmentLogValue(fspec.Compartment),
					"subcategory":  stringOrAny(fspec.Subcategory),
					"class_filter": fspec.Class,
				}).Debugf("Item-string search served.")
			}

			rms := make([]StringSearchResultRestModel, 0, len(rows))
			for _, row := range rows {
				rms = append(rms, StringSearchResultRestModel{
					Id:          strconv.Itoa(int(row.ItemId)),
					Name:        row.Name,
					Compartment: Compartment(row.Compartment).String(),
					Subcategory: row.Subcategory,
				})
			}

			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[[]StringSearchResultRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rms)
		}
	}
}

func buildPredicates(f filterSpec, isDefaultBrowse bool) ([]string, []interface{}) {
	var preds []string
	var args []interface{}

	if f.Compartment != nil {
		preds = append(preds, "compartment = ?")
		args = append(args, int(*f.Compartment))
	}
	if f.Subcategory != "" {
		preds = append(preds, "subcategory = ?")
		args = append(args, f.Subcategory)
	}
	if f.Class != "" {
		if f.ClassIsAny {
			preds = append(preds, "(job_mask IS NOT NULL AND job_mask = 0)")
		} else {
			preds = append(preds, "(job_mask IS NOT NULL AND (job_mask = 0 OR (job_mask & ?) = ?))")
			args = append(args, f.JobMaskBits, f.JobMaskBits)
		}
	}
	if isDefaultBrowse {
		preds = append(preds, "compartment != 0")
	}
	return preds, args
}

func compartmentLogValue(c *Compartment) string {
	if c == nil {
		return "any"
	}
	return c.String()
}

func stringOrAny(s string) string {
	if s == "" {
		return "any"
	}
	return s
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
