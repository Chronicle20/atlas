package quest

import (
	"atlas-quest/quest/progress"
	"atlas-quest/rest"
	"errors"
	"net/http"
	"strconv"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const (
	GetQuestsByCharacter          = "get_quests_by_character"
	GetQuestByCharacterAndId      = "get_quest_by_character_and_id"
	StartQuest                    = "start_quest"
	CompleteQuest                 = "complete_quest"
	ForfeitQuest                  = "forfeit_quest"
	GetQuestProgress              = "get_quest_progress"
	UpdateQuestProgress           = "update_quest_progress"
	GetStartedQuestsByCharacter   = "get_started_quests_by_character"
	GetCompletedQuestsByCharacter = "get_completed_quests_by_character"
	DeleteQuestsByCharacter       = "delete_quests_by_character"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(db)(si)

			// Character quest routes
			r := router.PathPrefix("/characters/{characterId}/quests").Subrouter()
			r.HandleFunc("", registerGet(GetQuestsByCharacter, handleGetQuestsByCharacter(db))).Methods(http.MethodGet)
			r.HandleFunc("", registerGet(DeleteQuestsByCharacter, handleDeleteQuestsByCharacter(db))).Methods(http.MethodDelete)
			r.HandleFunc("/started", registerGet(GetStartedQuestsByCharacter, handleGetQuestsByCharacterAndState(db, StateStarted))).Methods(http.MethodGet)
			r.HandleFunc("/completed", registerGet(GetCompletedQuestsByCharacter, handleGetQuestsByCharacterAndState(db, StateCompleted))).Methods(http.MethodGet)
			r.HandleFunc("/{questId}", registerGet(GetQuestByCharacterAndId, handleGetQuestByCharacterAndId(db))).Methods(http.MethodGet)
			r.HandleFunc("/{questId}/start", rest.RegisterInputHandler[StartQuestInputRestModel](l)(db)(si)(StartQuest, handleStartQuest)).Methods(http.MethodPost)
			r.HandleFunc("/{questId}/complete", rest.RegisterInputHandler[CompleteQuestInputRestModel](l)(db)(si)(CompleteQuest, handleCompleteQuest)).Methods(http.MethodPost)
			r.HandleFunc("/{questId}/forfeit", registerGet(ForfeitQuest, handleForfeitQuest(db))).Methods(http.MethodPost)
			r.HandleFunc("/{questId}/progress", registerGet(GetQuestProgress, handleGetQuestProgress(db))).Methods(http.MethodGet)
			r.HandleFunc("/{questId}/progress", rest.RegisterInputHandler[progress.RestModel](l)(db)(si)(UpdateQuestProgress, handleUpdateQuestProgress)).Methods(http.MethodPatch)
		}
	}
}

func handleGetQuestsByCharacter(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				quests, err := NewProcessor(d.Logger(), d.Context(), db).GetByCharacterId(characterId)
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to get quests for character [%d].", characterId)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				res, err := model.SliceMap(Transform)(model.FixedProvider(quests))()()
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				server.Marshal[[]RestModel](d.Logger())(w)(c.ServerInformation())(res)
			}
		})
	}
}

func handleGetQuestsByCharacterAndState(db *gorm.DB, state State) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				quests, err := NewProcessor(d.Logger(), d.Context(), db).GetByCharacterIdAndState(characterId, state)
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to get quests for character [%d] with state [%d].", characterId, state)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				res, err := model.SliceMap(Transform)(model.FixedProvider(quests))()()
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				server.Marshal[[]RestModel](d.Logger())(w)(c.ServerInformation())(res)
			}
		})
	}
}

func handleGetQuestByCharacterAndId(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return rest.ParseQuestId(d.Logger(), func(questId uint32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					q, err := NewProcessor(d.Logger(), d.Context(), db).GetByCharacterIdAndQuestId(characterId, questId)
					if errors.Is(err, gorm.ErrRecordNotFound) {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					if err != nil {
						d.Logger().WithError(err).Errorf("Unable to get quest [%d] for character [%d].", questId, characterId)
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					res, err := model.Map(Transform)(model.FixedProvider(q))()
					if err != nil {
						d.Logger().WithError(err).Errorf("Creating REST model.")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					server.Marshal[RestModel](d.Logger())(w)(c.ServerInformation())(res)
				}
			})
		})
	}
}

func handleStartQuest(d *rest.HandlerDependency, c *rest.HandlerContext, i StartQuestInputRestModel) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return rest.ParseQuestId(d.Logger(), func(questId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				f := field.NewBuilder(i.WorldId, i.ChannelId, i.MapId).Build()
				// REST endpoints use uuid.Nil since they're not saga-initiated
				q, failedConditions, err := NewProcessor(d.Logger(), d.Context(), d.DB()).Start(uuid.Nil, characterId, questId, f, i.SkipValidation)
				if errors.Is(err, ErrStartRequirementsNotMet) {
					// Return 422 Unprocessable Entity with failed conditions
					result := ValidationFailedRestModel{FailedConditions: failedConditions}
					w.WriteHeader(http.StatusUnprocessableEntity)
					server.Marshal[ValidationFailedRestModel](d.Logger())(w)(c.ServerInformation())(result)
					return
				}
				if errors.Is(err, ErrQuestAlreadyStarted) {
					w.WriteHeader(http.StatusConflict)
					return
				}
				if errors.Is(err, ErrQuestAlreadyCompleted) {
					w.WriteHeader(http.StatusConflict)
					return
				}
				if errors.Is(err, ErrIntervalNotElapsed) {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to start quest [%d] for character [%d].", questId, characterId)
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				res, err := model.Map(Transform)(model.FixedProvider(q))()
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				server.Marshal[RestModel](d.Logger())(w)(c.ServerInformation())(res)
			}
		})
	})
}

func handleCompleteQuest(d *rest.HandlerDependency, c *rest.HandlerContext, i CompleteQuestInputRestModel) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return rest.ParseQuestId(d.Logger(), func(questId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				f := field.NewBuilder(i.WorldId, i.ChannelId, i.MapId).Build()
				// REST endpoints use uuid.Nil since they're not saga-initiated
				nextQuestId, err := NewProcessor(d.Logger(), d.Context(), d.DB()).Complete(uuid.Nil, characterId, questId, f, i.SkipValidation)
				if errors.Is(err, gorm.ErrRecordNotFound) {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				if errors.Is(err, ErrQuestExpired) {
					w.WriteHeader(http.StatusGone) // 410 Gone for expired quests
					return
				}
				if errors.Is(err, ErrQuestNotStarted) {
					w.WriteHeader(http.StatusConflict) // 409 Conflict
					return
				}
				if errors.Is(err, ErrEndRequirementsNotMet) {
					w.WriteHeader(http.StatusUnprocessableEntity) // 422 for requirements not met
					return
				}
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to complete quest [%d] for character [%d].", questId, characterId)
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				// Return next quest ID in response if this is part of a chain
				if nextQuestId > 0 {
					result := CompleteQuestResponseRestModel{NextQuestId: nextQuestId}
					server.Marshal[CompleteQuestResponseRestModel](d.Logger())(w)(c.ServerInformation())(result)
					return
				}

				w.WriteHeader(http.StatusNoContent)
			}
		})
	})
}

func handleForfeitQuest(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return rest.ParseQuestId(d.Logger(), func(questId uint32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					// REST endpoints use uuid.Nil since they're not saga-initiated
					err := NewProcessor(d.Logger(), d.Context(), db).Forfeit(uuid.Nil, characterId, questId)
					if errors.Is(err, gorm.ErrRecordNotFound) {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					if err != nil {
						d.Logger().WithError(err).Errorf("Unable to forfeit quest [%d] for character [%d].", questId, characterId)
						w.WriteHeader(http.StatusBadRequest)
						return
					}

					w.WriteHeader(http.StatusNoContent)
				}
			})
		})
	}
}

func handleGetQuestProgress(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return rest.ParseQuestId(d.Logger(), func(questId uint32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					q, err := NewProcessor(d.Logger(), d.Context(), db).GetByCharacterIdAndQuestId(characterId, questId)
					if errors.Is(err, gorm.ErrRecordNotFound) {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					if err != nil {
						d.Logger().WithError(err).Errorf("Unable to get quest [%d] for character [%d].", questId, characterId)
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					res, err := model.SliceMap(progress.Transform)(model.FixedProvider(q.Progress()))()()
					if err != nil {
						d.Logger().WithError(err).Errorf("Creating REST model.")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					server.Marshal[[]progress.RestModel](d.Logger())(w)(c.ServerInformation())(res)
				}
			})
		})
	}
}

func handleUpdateQuestProgress(d *rest.HandlerDependency, _ *rest.HandlerContext, i progress.RestModel) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return rest.ParseQuestId(d.Logger(), func(questId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				// Check if infoNumber is provided in path or use from body
				infoNumberStr := mux.Vars(r)["infoNumber"]
				var infoNumber uint32
				if infoNumberStr != "" {
					val, err := strconv.Atoi(infoNumberStr)
					if err != nil {
						d.Logger().WithError(err).Errorf("Unable to parse infoNumber from path.")
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					infoNumber = uint32(val)
				} else {
					infoNumber = i.InfoNumber
				}

				// REST endpoints use uuid.Nil since they're not saga-initiated
				err := NewProcessor(d.Logger(), d.Context(), d.DB()).SetProgress(uuid.Nil, characterId, questId, infoNumber, i.Progress)
				if errors.Is(err, gorm.ErrRecordNotFound) {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to update progress for quest [%d] for character [%d].", questId, characterId)
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				w.WriteHeader(http.StatusNoContent)
			}
		})
	})
}

func handleDeleteQuestsByCharacter(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				err := NewProcessor(d.Logger(), d.Context(), db).DeleteByCharacterId(characterId)
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to delete quests for character [%d].", characterId)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				w.WriteHeader(http.StatusNoContent)
			}
		})
	}
}
