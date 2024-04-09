package rest

import (
	"context"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"net/http"
	"sync"
)

type RouteInitializer func(*mux.Router, logrus.FieldLogger, *gorm.DB)

func CreateService(l *logrus.Logger, db *gorm.DB, ctx context.Context, wg *sync.WaitGroup, basePath string, initializers ...RouteInitializer) {
	go NewServer(l, ctx, wg, ProduceRoutes(db, basePath, initializers...))
}

func ProduceRoutes(db *gorm.DB, basePath string, initializers ...RouteInitializer) func(l logrus.FieldLogger) http.Handler {
	return func(l logrus.FieldLogger) http.Handler {
		router := mux.NewRouter().PathPrefix(basePath).Subrouter().StrictSlash(true)
		router.Use(CommonHeader)

		for _, initializer := range initializers {
			initializer(router, l, db)
		}

		return router
	}
}
