package account

import (
	account2 "atlas-account/kafka/message/account"
	"atlas-account/rest"
	"errors"
	"net/http"
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

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
			registerCreateInput := rest.RegisterInputHandler[CreateRestModel](l)(si)
			registerPinAttemptInput := rest.RegisterInputHandler[PinAttemptInputRestModel](l)(si)
			registerPicAttemptInput := rest.RegisterInputHandler[PicAttemptInputRestModel](l)(si)

			r := router.PathPrefix("/accounts").Subrouter()
			r.HandleFunc("/", registerCreateInput("create_account", handleCreateAccount)).Methods(http.MethodPost)
			r.HandleFunc("/", register("get_account_by_name", handleGetAccountByName(db))).Queries("name", "{name}").Methods(http.MethodGet)
			r.HandleFunc("/", register("get_accounts", handleGetAccounts(db))).Methods(http.MethodGet)
			r.HandleFunc("/{accountId}", register("get_account", handleGetAccountById(db))).Methods(http.MethodGet)
			r.HandleFunc("/{accountId}", registerInput("update_account", handleUpdateAccount(db))).Methods(http.MethodPatch)
			r.HandleFunc("/{accountId}", register("delete_account", handleDeleteAccount(db))).Methods(http.MethodDelete)
			r.HandleFunc("/{accountId}/pin-attempts", registerPinAttemptInput("record_pin_attempt", handleRecordPinAttempt(db))).Methods(http.MethodPost)
			r.HandleFunc("/{accountId}/pic-attempts", registerPicAttemptInput("record_pic_attempt", handleRecordPicAttempt(db))).Methods(http.MethodPost)
			r.HandleFunc("/{accountId}/session", register("delete_account_session", handleDeleteAccountSession)).Methods(http.MethodDelete)
			r.HandleFunc("/{accountId}/worlds/{worldId}/character-slots", register("get_account_character_slots", handleGetCharacterSlots(db))).Methods(http.MethodGet)
			r.HandleFunc("/{accountId}/worlds/{worldId}/character-slots", register("increment_account_character_slots", handleIncrementCharacterSlots(db))).Methods(http.MethodPost)
		}
	}
}

func handleUpdateAccount(db *gorm.DB) rest.InputHandler[RestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
		return rest.ParseAccountId(d.Logger(), func(accountId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				im, err := model.Map(Extract)(model.FixedProvider(input))()
				if err != nil {
					d.Logger().WithError(err).Errorf("Invalid input.")
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				a, err := NewProcessor(d.Logger(), d.Context(), db).Update(accountId, im)
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to update account [%d].", accountId)
					w.WriteHeader(http.StatusNotFound)
					return
				}

				res, err := model.Map(Transform)(model.FixedProvider(a))()
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

func handleCreateAccount(d *rest.HandlerDependency, _ *rest.HandlerContext, input CreateRestModel) http.HandlerFunc {
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

func handleGetAccountByName(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return parseName(d.Logger(), func(name string) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				res, err := model.Map(Transform)(NewProcessor(d.Logger(), d.Context(), db).ByNameProvider(name))()
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
}

func handleGetAccounts(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			page, err := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
			if err != nil {
				server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
				return
			}

			paged, err := NewProcessor(d.Logger(), d.Context(), db).AllProvider(page)()
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to locate accounts.")
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

func handleGetAccountById(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseAccountId(d.Logger(), func(id uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				res, err := model.Map(Transform)(NewProcessor(d.Logger(), d.Context(), db).ByIdProvider(id))()
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
}

func handleDeleteAccount(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, _ *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseAccountId(d.Logger(), func(accountId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				err := NewProcessor(d.Logger(), d.Context(), db).DeleteAndEmit(accountId)
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
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			}
		})
	}
}

func handleRecordPinAttempt(db *gorm.DB) rest.InputHandler[PinAttemptInputRestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, input PinAttemptInputRestModel) http.HandlerFunc {
		return rest.ParseAccountId(d.Logger(), func(accountId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				attempts, limitReached, err := NewProcessor(d.Logger(), d.Context(), db).RecordPinAttemptAndEmit(accountId, input.Success, input.IpAddress, input.HWID)
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to record PIN attempt for account [%d].", accountId)
					server.WriteErrorResponse(d.Logger())(w)(err)
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
}

func handleRecordPicAttempt(db *gorm.DB) rest.InputHandler[PicAttemptInputRestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, input PicAttemptInputRestModel) http.HandlerFunc {
		return rest.ParseAccountId(d.Logger(), func(accountId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				attempts, limitReached, err := NewProcessor(d.Logger(), d.Context(), db).RecordPicAttemptAndEmit(accountId, input.Success, input.IpAddress, input.HWID)
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to record PIC attempt for account [%d].", accountId)
					server.WriteErrorResponse(d.Logger())(w)(err)
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
}

func handleDeleteAccountSession(d *rest.HandlerDependency, _ *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseAccountId(d.Logger(), func(accountId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			_ = producer.ProviderImpl(d.Logger())(d.Context())(account2.EnvCommandSessionTopic)(logoutCommandProvider(accountId))
			w.WriteHeader(http.StatusAccepted)
		}
	})
}

func handleGetCharacterSlots(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseAccountIdAndWorldId(d.Logger(), func(accountId uint32, worldId byte) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				cs, err := NewProcessor(d.Logger(), d.Context(), db).GetCharacterSlots(accountId, worldId)
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to retrieve character slots for account [%d] in world [%d].", accountId, worldId)
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				res, err := model.Map(TransformCharacterSlot)(model.FixedProvider(cs))()
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[CharacterSlotRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
			}
		})
	}
}

func handleIncrementCharacterSlots(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseAccountIdAndWorldId(d.Logger(), func(accountId uint32, worldId byte) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				cs, err := NewProcessor(d.Logger(), d.Context(), db).IncrementCharacterSlots(accountId, worldId)
				if err != nil {
					if errors.Is(err, ErrCharacterSlotCapReached) {
						w.WriteHeader(http.StatusConflict)
						return
					}
					d.Logger().WithError(err).Errorf("Unable to increment character slots for account [%d] in world [%d].", accountId, worldId)
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				res, err := model.Map(TransformCharacterSlot)(model.FixedProvider(cs))()
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[CharacterSlotRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
			}
		})
	}
}
