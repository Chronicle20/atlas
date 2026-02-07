package ban

import (
	"atlas-ban/rest"
	"net/http"
	"strconv"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			register := rest.RegisterHandler(l)(db)(si)
			registerInput := rest.RegisterInputHandler[RestModel](l)(db)(si)

			r := router.PathPrefix("/bans").Subrouter()
			r.HandleFunc("/", registerInput("create_ban", handleCreateBan)).Methods(http.MethodPost)
			r.HandleFunc("/", register("get_bans", handleGetBans)).Methods(http.MethodGet)
			r.HandleFunc("/check", register("check_ban", handleCheckBan)).Methods(http.MethodGet)
			r.HandleFunc("/{banId}", register("get_ban", handleGetBanById)).Methods(http.MethodGet)
			r.HandleFunc("/{banId}", register("delete_ban", handleDeleteBan)).Methods(http.MethodDelete)
		}
	}
}

func handleCreateBan(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m, err := NewProcessor(d.Logger(), d.Context(), d.DB()).CreateAndEmit(
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
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res, err := Transform(m)
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating REST model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		w.WriteHeader(http.StatusCreated)
		server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
	}
}

func handleGetBans(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		banTypeStr := r.URL.Query().Get("type")

		var bans []Model
		var err error

		if banTypeStr != "" {
			bt, parseErr := strconv.Atoi(banTypeStr)
			if parseErr != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			bans, err = NewProcessor(d.Logger(), d.Context(), d.DB()).GetByType(BanType(bt))
		} else {
			bans, err = NewProcessor(d.Logger(), d.Context(), d.DB()).GetByTenant()
		}

		if err != nil {
			d.Logger().WithError(err).Errorf("Unable to locate bans.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res, err := model.SliceMap(Transform)(model.FixedProvider(bans))(model.ParallelMap())()
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

func handleGetBanById(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseBanId(d.Logger(), func(banId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			m, err := NewProcessor(d.Logger(), d.Context(), d.DB()).GetById(banId)
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to retrieve ban [%d].", banId)
				w.WriteHeader(http.StatusNotFound)
				return
			}

			res, err := Transform(m)
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	})
}

func handleDeleteBan(d *rest.HandlerDependency, _ *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseBanId(d.Logger(), func(banId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			err := NewProcessor(d.Logger(), d.Context(), d.DB()).DeleteAndEmit(banId)
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to delete ban [%d].", banId)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		}
	})
}

func handleCheckBan(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
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

		m, err := NewProcessor(d.Logger(), d.Context(), d.DB()).CheckBan(ip, hwid, accountId)
		if err != nil {
			d.Logger().WithError(err).Errorf("Unable to check ban for ip [%s] hwid [%s] account [%d].", ip, hwid, accountId)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res := TransformCheck(m)

		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalResponse[CheckRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
	}
}
