package character

import (
	"net/http"

	"atlas-renders/storage"

	"github.com/sirupsen/logrus"
)

// Handler is the character composite render endpoint.
//
// TODO Task 13 follow-up: port donor characterrender/{handler,hash,
// query,path,write,resource,error,otel}.go logic onto Storage.GetAtlas
// + bytes blits. Until then, surface the gap explicitly.
func Handler(l logrus.FieldLogger, s *storage.Storage) http.HandlerFunc {
	_ = s
	return func(w http.ResponseWriter, r *http.Request) {
		l.Warn("character.Handler: not yet implemented; see task-071 follow-up")
		http.Error(w, "character render not yet implemented", http.StatusNotImplemented)
	}
}
