package main

import (
	"atlas-data/equipment/slots"
	"atlas-data/equipment/statistics"
	"atlas-data/logger"
	_map "atlas-data/map"
	"atlas-data/monster"
	"atlas-data/service"
	"atlas-data/tracing"
	"atlas-data/wz"
	"github.com/Chronicle20/atlas-rest/server"
	"os"
)

const serviceName = "atlas-data"

type Server struct {
	baseUrl string
	prefix  string
}

func (s Server) GetBaseURL() string {
	return s.baseUrl
}

func (s Server) GetPrefix() string {
	return s.prefix
}

func GetServer() Server {
	return Server{
		baseUrl: "",
		prefix:  "/api/gis/",
	}
}

func main() {
	l := logger.CreateLogger(serviceName)
	l.Infoln("Starting main service.")

	tdm := service.GetTeardownManager()

	tc, err := tracing.InitTracer(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	wzDir := os.Getenv("GAME_DATA_ROOT_DIR")

	l.Infoln("Initializing game data cache.")
	wz.GetFileCache().Init(wzDir)
	l.Infoln("Completed initializing game data cache.")

	server.CreateService(l, tdm.Context(), tdm.WaitGroup(), GetServer().GetPrefix(),
		_map.InitResource(GetServer()),
		monster.InitResource(GetServer()),
		slots.InitResource(GetServer()),
		statistics.InitResource(GetServer()))

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
