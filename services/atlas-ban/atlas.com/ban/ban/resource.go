package ban

import (
	"atlas-ban/rest"
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			register := rest.RegisterHandler(l)(si)
			registerInput := rest.RegisterInputHandler[RestModel](l)(si)

			r := router.PathPrefix("/bans").Subrouter()
			r.HandleFunc("/", registerInput("create_ban", handleCreateBan(db))).Methods(http.MethodPost)
			r.HandleFunc("/", register("get_bans", handleGetBans(db))).Methods(http.MethodGet)
			r.HandleFunc("/check", register("check_ban", handleCheckBan(db))).Methods(http.MethodGet)
			r.HandleFunc("/{banId}", register("get_ban", handleGetBanById(db))).Methods(http.MethodGet)
			r.HandleFunc("/{banId}", register("delete_ban", handleDeleteBan(db))).Methods(http.MethodDelete)
			r.HandleFunc("/{banId}/expire", register("expire_ban", handleExpireBan(db))).Methods(http.MethodPost)
		}
	}
}

func handleCreateBan(db *gorm.DB) rest.InputHandler[RestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			m, err := NewProcessor(d.Logger(), d.Context(), db).CreateAndEmit(
				BanType(input.BanType),
				input.Value,
				input.Reason,
				input.ReasonCode,
				input.Permanent,
				input.ExpiresAt,
				input.IssuedBy,
			)
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to create ban.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			res, err := Transform(m)
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			w.WriteHeader(http.StatusCreated)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	}
}

func handleGetBans(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			banTypeStr := r.URL.Query().Get("type")

			if banTypeStr != "" {
				bt, parseErr := strconv.Atoi(banTypeStr)
				if parseErr != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				page, err := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
				if err != nil {
					server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
					return
				}

				paged, err := NewProcessor(d.Logger(), d.Context(), db).ByTypePagedProvider(BanType(bt), page)()
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to locate bans.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				res, err := model.SliceMap(Transform)(model.FixedProvider(paged.Items))(model.ParallelMap())()
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res, paginate.EnvelopeFor(paged), r)
				return
			}

			page, err := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
			if err != nil {
				server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
				return
			}

			paged, err := NewProcessor(d.Logger(), d.Context(), db).AllProvider(page)()
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to locate bans.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			res, err := model.SliceMap(Transform)(model.FixedProvider(paged.Items))(model.ParallelMap())()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res, paginate.EnvelopeFor(paged), r)
		}
	}
}

func handleGetBanById(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseBanId(d.Logger(), func(banId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				m, err := NewProcessor(d.Logger(), d.Context(), db).GetById(banId)
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to retrieve ban [%d].", banId)
					w.WriteHeader(http.StatusNotFound)
					return
				}

				res, err := Transform(m)
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

func handleDeleteBan(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, _ *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseBanId(d.Logger(), func(banId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				err := NewProcessor(d.Logger(), d.Context(), db).DeleteAndEmit(banId)
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to delete ban [%d].", banId)
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			}
		})
	}
}

func handleExpireBan(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, _ *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseBanId(d.Logger(), func(banId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				err := NewProcessor(d.Logger(), d.Context(), db).ExpireBanAndEmit(banId)
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to expire ban [%d].", banId)
					if errors.Is(err, ErrCannotExpirePermanentBan) {
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			}
		})
	}
}

func handleCheckBan(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ip := r.URL.Query().Get("ip")
			hwid := r.URL.Query().Get("hwid")
			accountIdStr := r.URL.Query().Get("accountId")

			var accountId uint32
			if accountIdStr != "" {
				v, err := strconv.ParseUint(accountIdStr, 10, 32)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				accountId = uint32(v)
			}

			m, err := NewProcessor(d.Logger(), d.Context(), db).CheckBan(ip, hwid, accountId)
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to check ban for ip [%s] hwid [%s] account [%d].", ip, hwid, accountId)
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			res := TransformCheck(m)

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[CheckRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	}
}
