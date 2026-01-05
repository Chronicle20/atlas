package quest

import (
	"atlas-quest/quest/progress"
	"atlas-quest/rest"
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

const (
	GetQuestsByCharacter       = "get_quests_by_character"
	GetQuestByCharacterAndId   = "get_quest_by_character_and_id"
	StartQuest                 = "start_quest"
	CompleteQuest              = "complete_quest"
	ForfeitQuest               = "forfeit_quest"
	GetQuestProgress           = "get_quest_progress"
	UpdateQuestProgress        = "update_quest_progress"
	GetStartedQuestsByCharacter = "get_started_quests_by_character"
	GetCompletedQuestsByCharacter = "get_completed_quests_by_character"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(db)(si)

			// Character quest routes
			r := router.PathPrefix("/characters/{characterId}/quests").Subrouter()
			r.HandleFunc("", registerGet(GetQuestsByCharacter, handleGetQuestsByCharacter(db))).Methods(http.MethodGet)
			r.HandleFunc("/started", registerGet(GetStartedQuestsByCharacter, handleGetQuestsByCharacterAndState(db, StateStarted))).Methods(http.MethodGet)
			r.HandleFunc("/completed", registerGet(GetCompletedQuestsByCharacter, handleGetQuestsByCharacterAndState(db, StateCompleted))).Methods(http.MethodGet)
			r.HandleFunc("/{questId}", registerGet(GetQuestByCharacterAndId, handleGetQuestByCharacterAndId(db))).Methods(http.MethodGet)
			r.HandleFunc("/{questId}/start", registerGet(StartQuest, handleStartQuest(db))).Methods(http.MethodPost)
			r.HandleFunc("/{questId}/complete", registerGet(CompleteQuest, handleCompleteQuest(db))).Methods(http.MethodPost)
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

func handleStartQuest(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return rest.ParseQuestId(d.Logger(), func(questId uint32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					q, err := NewProcessor(d.Logger(), d.Context(), db).Start(characterId, questId)
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
}

func handleCompleteQuest(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return rest.ParseQuestId(d.Logger(), func(questId uint32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					err := NewProcessor(d.Logger(), d.Context(), db).Complete(characterId, questId)
					if errors.Is(err, gorm.ErrRecordNotFound) {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					if err != nil {
						d.Logger().WithError(err).Errorf("Unable to complete quest [%d] for character [%d].", questId, characterId)
						w.WriteHeader(http.StatusBadRequest)
						return
					}

					w.WriteHeader(http.StatusNoContent)
				}
			})
		})
	}
}

func handleForfeitQuest(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return rest.ParseQuestId(d.Logger(), func(questId uint32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					err := NewProcessor(d.Logger(), d.Context(), db).Forfeit(characterId, questId)
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

func handleUpdateQuestProgress(d *rest.HandlerDependency, c *rest.HandlerContext, i progress.RestModel) http.HandlerFunc {
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

				err := NewProcessor(d.Logger(), d.Context(), d.DB()).SetProgress(characterId, questId, infoNumber, i.Progress)
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
