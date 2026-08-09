package ledger

import (
	"atlas-trades/rest"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
)

const (
	GetLedgerEntries   = "get_trade_ledger_entries"
	GetLedgerEntryById = "get_trade_ledger_entry_by_id"
)

// maxPageSize is PRD §5's page[size] cap for the ledger list: a request above
// it is a 400, never a silent clamp. It is lower than the repo-wide
// paginate.MaxPageSize (250) because the PRD sets it; the endpoint's DEFAULT is
// separately paginate.DefaultPageSize, passed at the ParseParams call below.
const maxPageSize = 100

// The default window when filter[from] / filter[to] are absent: everything ever
// recorded. The upper bound is a fixed far-future timestamp rather than
// time.Now() so an entry whose settled_at is marginally ahead of the reader's
// clock — the settlement handler and the reader are different pods — is still
// returned.
var (
	defaultFrom = time.Time{}
	defaultTo   = time.Date(9999, time.December, 31, 23, 59, 59, 0, time.UTC)
)

// InitResource wires PRD §5's two read-only ledger endpoints:
// GET /trades/ledger and GET /trades/ledger/{entryId}. The db handle is curried
// in at wiring time so each handler can construct the tenant-scoped Processor.
// There are no write routes: ledger rows are immutable (FR-7.4).
func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(si)

			r := router.PathPrefix("/trades/ledger").Subrouter()
			r.HandleFunc("", registerGet(GetLedgerEntries, handleGetLedgerEntries(db))).Methods(http.MethodGet)
			r.HandleFunc("/{entryId}", registerGet(GetLedgerEntryById, handleGetLedgerEntryById(db))).Methods(http.MethodGet)
		}
	}
}

// ledgerFilters is PRD §5's filter[...] set for GET /trades/ledger.
//
// characterId is REQUIRED. FR-7.2 defines the ledger read as a per-character
// lookup and the ledger has no unfiltered provider, so a request without it is
// a client error rather than a whole-table scan.
type ledgerFilters struct {
	characterId character.Id
	from        time.Time
	to          time.Time
}

// parseLedgerFilters reads filter[characterId] and the optional RFC3339
// filter[from] / filter[to] window. Anything unparseable is an error rather
// than a silently dropped filter, which would answer a narrow question with a
// wider slice of the ledger.
func parseLedgerFilters(query url.Values) (ledgerFilters, error) {
	raw := query.Get("filter[characterId]")
	if raw == "" {
		return ledgerFilters{}, errors.New("filter[characterId] is required")
	}
	id, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		return ledgerFilters{}, err
	}

	f := ledgerFilters{characterId: character.Id(id), from: defaultFrom, to: defaultTo}
	if f.from, err = parseTimeFilter(query, "filter[from]", defaultFrom); err != nil {
		return ledgerFilters{}, err
	}
	if f.to, err = parseTimeFilter(query, "filter[to]", defaultTo); err != nil {
		return ledgerFilters{}, err
	}
	return f, nil
}

// parseTimeFilter reads one optional RFC3339 timestamp filter, falling back to
// the supplied default when it is absent.
func parseTimeFilter(query url.Values, name string, fallback time.Time) (time.Time, error) {
	raw := query.Get(name)
	if raw == "" {
		return fallback, nil
	}
	return time.Parse(time.RFC3339, raw)
}

// handleGetLedgerEntries serves GET /trades/ledger — every settled trade in the
// window on which the character appears as either side (FR-7.2), newest first
// and paged.
func handleGetLedgerEntries(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query()

			filters, err := parseLedgerFilters(query)
			if err != nil {
				server.WriteBadRequest(d.Logger(), w, "invalid filter[characterId]/filter[from]/filter[to]; filter[characterId] is required and filter[from]/filter[to] are RFC3339")
				return
			}

			page, err := paginate.ParseParams(query, paginate.DefaultPageSize, maxPageSize)
			if err != nil {
				server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
				return
			}

			// Paged in SQL: the ledger grows without bound, so the page must
			// never be cut out of a fully materialised history.
			paged, err := NewProcessor(d.Logger(), d.Context(), db).GetPageByCharacterId(filters.characterId, filters.from, filters.to, page)
			if err != nil {
				d.Logger().WithError(err).Errorf("Reading ledger entries for character [%d].", filters.characterId)
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			res, err := model.SliceMap(Transform)(model.FixedProvider(paged.Items))(model.ParallelMap())()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res, paginate.EnvelopeFor(paged), r)
		}
	}
}

// handleGetLedgerEntryById serves GET /trades/ledger/{entryId}. The read is
// tenant-scoped, so another tenant's entry is a 404 rather than a leak.
func handleGetLedgerEntryById(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseEntryId(d.Logger(), func(entryId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				entry, err := NewProcessor(d.Logger(), d.Context(), db).GetById(entryId)
				if errors.Is(err, gorm.ErrRecordNotFound) {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				if err != nil {
					d.Logger().WithError(err).Errorf("Reading ledger entry [%s].", entryId)
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				res, err := Transform(entry)
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
			}
		})
	}
}
