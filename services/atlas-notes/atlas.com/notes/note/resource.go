package note

import (
	"atlas-notes/rest"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"net/http"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerHandler := rest.RegisterHandler(l)(db)(si)
			registerInputHandler := rest.RegisterInputHandler[RestModel](l)(db)(si)

			// ByIdProvider all notes
			router.HandleFunc("/notes", registerHandler("get_all_notes", GetAllNotesHandler)).Methods(http.MethodGet)

			// ByIdProvider all notes for a character
			router.HandleFunc(
				"/characters/{"+characterIdPattern+"}/notes",
				registerHandler("get_character_notes", GetCharacterNotesHandler),
			).Methods(http.MethodGet)

			// ByIdProvider a specific note
			router.HandleFunc(
				"/notes/{"+noteIdPattern+"}",
				registerHandler("get_note", GetNoteHandler),
			).Methods(http.MethodGet)

			// Create a note
			router.HandleFunc("/notes", registerInputHandler("create_note", CreateNoteHandler)).Methods(http.MethodPost)

			// Update a note
			router.HandleFunc(
				"/notes/{"+noteIdPattern+"}",
				registerInputHandler("update_note", UpdateNoteHandler),
			).Methods(http.MethodPatch)

			// Delete a note
			router.HandleFunc(
				"/notes/{"+noteIdPattern+"}",
				registerHandler("delete_note", DeleteNoteHandler),
			).Methods(http.MethodDelete)

			// Delete all notes for a character
			router.HandleFunc(
				"/characters/{"+characterIdPattern+"}/notes",
				registerHandler("delete_character_notes", DeleteCharacterNotesHandler),
			).Methods(http.MethodDelete)
		}
	}
}

// GetAllNotesHandler handles GET /api/notes
func GetAllNotesHandler(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mp := NewProcessor(d.Logger(), d.Context(), d.DB()).InTenantProvider()
		rm, err := model.SliceMap(Transform)(mp)(model.ParallelMap())()
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating REST model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
	}
}

// GetCharacterNotesHandler handles GET /api/characters/{characterId}/notes
func GetCharacterNotesHandler(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			mp := new(ProcessorImpl).ByCharacterProvider(characterId)
			rm, err := model.SliceMap(Transform)(mp)(model.ParallelMap())()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
		}
	})
}

// GetNoteHandler handles GET /api/notes/{noteId}
func GetNoteHandler(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseNoteId(d.Logger(), func(noteId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			mp := new(ProcessorImpl).ByIdProvider(noteId)
			rm, err := model.Map(Transform)(mp)()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
		}
	})
}

// CreateNoteHandler handles POST /api/notes
func CreateNoteHandler(d *rest.HandlerDependency, c *rest.HandlerContext, i RestModel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		im, err := Extract(i)
		if err != nil {
			d.Logger().WithError(err).Errorln("Error extracting note data")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		m, err := NewProcessor(d.Logger(), d.Context(), d.DB()).CreateAndEmit(im.CharacterId(), im.SenderId(), im.Message(), im.Flag())
		rm, err := model.Map(Transform)(model.FixedProvider(m))()
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating REST model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
	}
}

// UpdateNoteHandler handles PATCH /api/notes/{noteId}
func UpdateNoteHandler(d *rest.HandlerDependency, c *rest.HandlerContext, i RestModel) http.HandlerFunc {
	return rest.ParseNoteId(d.Logger(), func(noteId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			im, err := Extract(i)
			if err != nil {
				d.Logger().WithError(err).Errorln("Error extracting note data")
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			if noteId != im.Id() {
				d.Logger().Errorln("Note ID does not match URL")
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			m, err := NewProcessor(d.Logger(), d.Context(), d.DB()).UpdateAndEmit(im.Id(), im.CharacterId(), im.SenderId(), im.Message(), im.Flag())
			rm, err := model.Map(Transform)(model.FixedProvider(m))()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
		}
	})
}

// DeleteNoteHandler handles DELETE /api/notes/{noteId}
func DeleteNoteHandler(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseNoteId(d.Logger(), func(noteId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			err := NewProcessor(d.Logger(), d.Context(), d.DB()).DeleteAndEmit(noteId)
			if err != nil {
				d.Logger().WithError(err).Errorln("Error deleting note")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusNoContent)
		}
	})
}

// DeleteCharacterNotesHandler handles DELETE /api/characters/{characterId}/notes
func DeleteCharacterNotesHandler(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			err := NewProcessor(d.Logger(), d.Context(), d.DB()).DeleteAllAndEmit(characterId)
			if err != nil {
				d.Logger().WithError(err).Errorln("Error deleting character notes")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusNoContent)
		}
	})
}
