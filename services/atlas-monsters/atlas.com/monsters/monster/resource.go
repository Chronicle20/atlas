package monster

import (
	"atlas-monsters/rest"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/manyminds/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"net/http"
)

const (
	getMonster = "get_monster"
)

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		r := router.PathPrefix("/monsters").Subrouter()
		r.HandleFunc("/{monsterId}", rest.RegisterHandler(l)(si)(getMonster, handleGetMonsterById)).Methods(http.MethodGet)
	}
}

func handleGetMonsterById(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseMonsterId(d.Logger(), func(monsterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			m, err := GetById(c.Tenant())(monsterId)
			if err != nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			res, err := model.Map(model.FixedProvider(m), Transform)()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			server.Marshal[RestModel](d.Logger())(w)(c.ServerInformation())(res)
		}
	})
}
