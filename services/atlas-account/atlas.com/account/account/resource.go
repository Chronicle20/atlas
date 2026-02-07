package account

import (
	account2 "atlas-account/kafka/message/account"
	"atlas-account/kafka/producer"
	"atlas-account/rest"
	"errors"
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
			registerPinAttemptInput := rest.RegisterInputHandler[PinAttemptInputRestModel](l)(db)(si)
		registerPicAttemptInput := rest.RegisterInputHandler[PicAttemptInputRestModel](l)(db)(si)

			r := router.PathPrefix("/accounts").Subrouter()
			r.HandleFunc("/", registerInput("create_account", handleCreateAccount)).Methods(http.MethodPost)
			r.HandleFunc("/", register("get_account_by_name", handleGetAccountByName)).Queries("name", "{name}").Methods(http.MethodGet)
			r.HandleFunc("/", register("get_accounts", handleGetAccounts)).Methods(http.MethodGet)
			r.HandleFunc("/{accountId}", register("get_account", handleGetAccountById)).Methods(http.MethodGet)
			r.HandleFunc("/{accountId}", registerInput("update_account", handleUpdateAccount)).Methods(http.MethodPatch)
			r.HandleFunc("/{accountId}", register("delete_account", handleDeleteAccount)).Methods(http.MethodDelete)
			r.HandleFunc("/{accountId}/pin-attempts", registerPinAttemptInput("record_pin_attempt", handleRecordPinAttempt)).Methods(http.MethodPost)
			r.HandleFunc("/{accountId}/pic-attempts", registerPicAttemptInput("record_pic_attempt", handleRecordPicAttempt)).Methods(http.MethodPost)
			r.HandleFunc("/{accountId}/session", register("delete_account_session", handleDeleteAccountSession)).Methods(http.MethodDelete)
		}
	}
}

func handleUpdateAccount(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
	return rest.ParseAccountId(d.Logger(), func(accountId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			im, err := model.Map(Extract)(model.FixedProvider(input))()
			if err != nil {
				d.Logger().WithError(err).Errorf("Invalid input.")
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			a, err := NewProcessor(d.Logger(), d.Context(), d.DB()).Update(accountId, im)
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to update account [%d].", accountId)
				w.WriteHeader(http.StatusNotFound)
				return
			}

			res, err := model.Map(Transform)(model.FixedProvider(a))()
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

func handleCreateAccount(d *rest.HandlerDependency, _ *rest.HandlerContext, input RestModel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_ = producer.ProviderImpl(d.Logger())(d.Context())(account2.EnvCommandTopic)(createCommandProvider(input.Name, input.Password))
		w.WriteHeader(http.StatusAccepted)
	}
}

type nameHandler func(name string) http.HandlerFunc

func parseName(l logrus.FieldLogger, next nameHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if val, ok := mux.Vars(r)["name"]; ok {
			next(val)(w, r)
		} else {
			l.Errorf("Missing name parameter.")
			w.WriteHeader(http.StatusBadRequest)
		}
	}
}

func handleGetAccountByName(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return parseName(d.Logger(), func(name string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			res, err := model.Map(Transform)(NewProcessor(d.Logger(), d.Context(), d.DB()).ByNameProvider(name))()
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to retrieve account by name [%s].", name)
				w.WriteHeader(http.StatusNotFound)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	})
}

func handleGetAccounts(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		as, err := NewProcessor(d.Logger(), d.Context(), d.DB()).GetByTenant()
		if err != nil {
			d.Logger().WithError(err).Errorf("Unable to locate accounts.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res, err := model.SliceMap(Transform)(model.FixedProvider(as))(model.ParallelMap())()
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

func handleGetAccountById(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseAccountId(d.Logger(), func(id uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			res, err := model.Map(Transform)(NewProcessor(d.Logger(), d.Context(), d.DB()).ByIdProvider(id))()
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to locate account [%d].", id)
				w.WriteHeader(http.StatusNotFound)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	})
}

func handleDeleteAccount(d *rest.HandlerDependency, _ *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseAccountId(d.Logger(), func(accountId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			err := NewProcessor(d.Logger(), d.Context(), d.DB()).DeleteAndEmit(accountId)
			if err != nil {
				if errors.Is(err, ErrAccountNotFound) {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				if errors.Is(err, ErrAccountLoggedIn) {
					w.WriteHeader(http.StatusConflict)
					return
				}
				d.Logger().WithError(err).Errorf("Unable to delete account [%d].", accountId)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		}
	})
}

func handleRecordPinAttempt(d *rest.HandlerDependency, c *rest.HandlerContext, input PinAttemptInputRestModel) http.HandlerFunc {
	return rest.ParseAccountId(d.Logger(), func(accountId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			attempts, limitReached, err := NewProcessor(d.Logger(), d.Context(), d.DB()).RecordPinAttemptAndEmit(accountId, input.Success)
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to record PIN attempt for account [%d].", accountId)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			res := PinAttemptOutputRestModel{
				Id:           strconv.Itoa(int(accountId)),
				Attempts:     attempts,
				LimitReached: limitReached,
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[PinAttemptOutputRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	})
}

func handleRecordPicAttempt(d *rest.HandlerDependency, c *rest.HandlerContext, input PicAttemptInputRestModel) http.HandlerFunc {
	return rest.ParseAccountId(d.Logger(), func(accountId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			attempts, limitReached, err := NewProcessor(d.Logger(), d.Context(), d.DB()).RecordPicAttemptAndEmit(accountId, input.Success)
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to record PIC attempt for account [%d].", accountId)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			res := PicAttemptOutputRestModel{
				Id:           strconv.Itoa(int(accountId)),
				Attempts:     attempts,
				LimitReached: limitReached,
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[PicAttemptOutputRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	})
}

func handleDeleteAccountSession(d *rest.HandlerDependency, _ *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseAccountId(d.Logger(), func(accountId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			_ = producer.ProviderImpl(d.Logger())(d.Context())(account2.EnvCommandSessionTopic)(logoutCommandProvider(accountId))
			w.WriteHeader(http.StatusAccepted)
		}
	})
}
