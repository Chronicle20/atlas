package mapr

import (
	"net/http"

	"atlas-renders/storage"

	"github.com/sirupsen/logrus"
)

// Handler is the map composite render endpoint.
//
// TODO Task 13 follow-up: implement two paths per design §4.7:
//   - kind=minimap → 302 to /api/assets/.../minimap.png
//   - kind=render → MinIO probe + composite per layout.zmap, ported
//     from extractor's mapimage/{renderer,blit,sort}.go
func Handler(l logrus.FieldLogger, s *storage.Storage) http.HandlerFunc {
	_ = s
	return func(w http.ResponseWriter, r *http.Request) {
		l.Warn("mapr.Handler: not yet implemented; see task-071 follow-up")
		http.Error(w, "map render not yet implemented", http.StatusNotImplemented)
	}
}
