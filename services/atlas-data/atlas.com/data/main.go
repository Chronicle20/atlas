package main

import (
	"atlas-data/consumable"
	"atlas-data/data"
	"atlas-data/database"
	"atlas-data/equipment"
	data2 "atlas-data/kafka/consumer/data"
	"atlas-data/logger"
	_map "atlas-data/map"
	"atlas-data/monster"
	"atlas-data/pet"
	"atlas-data/reactor"
	"atlas-data/service"
	"atlas-data/skill"
	"atlas-data/tracing"
	"errors"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const serviceName = "atlas-data"
const consumerGroupId = "Data Service"

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
		prefix:  "/api/",
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

	db := database.Connect(l)

	//dir, exists := os.LookupEnv("GAME_DATA_ROOT_DIR")
	//if !exists {
	//	l.Errorf("Unable to retrieve [GAME_DATA_ROOT_DIR] configuration necessary to ingest data.")
	//	return
	//}

	//ts, err := collectUniqueFiles(dir)
	//if err != nil {
	//	l.WithError(err).Fatal("Unable to collect unique files.")
	//	return
	//}
	//for _, t := range ts {
	//	tctx := tenant.WithContext(context.Background(), t)
	//	_ = data.RegisterData(l)(tctx)
	//}

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	data2.InitConsumers(l)(cmf)(consumerGroupId)
	data2.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler)

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(data.InitResource(GetServer())).
		AddRouteInitializer(_map.InitResource(GetServer())).
		AddRouteInitializer(monster.InitResource(GetServer())).
		AddRouteInitializer(equipment.InitResource(GetServer())).
		AddRouteInitializer(reactor.InitResource(GetServer())).
		AddRouteInitializer(skill.InitResource(GetServer())).
		AddRouteInitializer(pet.InitResource(GetServer())).
		AddRouteInitializer(consumable.InitResource(GetServer())).
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}

func collectUniqueFiles(root string) ([]tenant.Model, error) {
	uniqueFiles := make([]tenant.Model, 0)

	// Helper function to recursively iterate up to three levels
	var walk func(path string, depth int) error
	walk = func(path string, depth int) error {
		if depth > 3 {
			return nil // Stop recursion beyond three levels
		}

		entries, err := os.ReadDir(path)
		if err != nil {
			return err // Handle errors accessing the directory
		}

		for _, entry := range entries {
			fullPath := filepath.Join(path, entry.Name())

			// If this is a directory, recurse
			if entry.IsDir() {
				if depth < 3 { // Only recurse for tenant, region, and version levels
					if err := walk(fullPath, depth+1); err != nil {
						return err
					}
				}
				if depth == 3 { // Process files only at the third level
					relativePath, err := filepath.Rel(root, fullPath)
					if err != nil {
						return err
					}

					parts := strings.Split(relativePath, string(os.PathSeparator))
					if len(parts) < 3 {
						continue // Skip files without enough levels
					}

					tid := uuid.MustParse(parts[0])
					region := parts[1]
					version := parts[2]
					versions := strings.Split(version, ".")
					if len(versions) != 2 {
						return errors.New("invalid folder structure")
					}
					mav, err := strconv.Atoi(versions[0])
					if err != nil {
						return err
					}
					miv, err := strconv.Atoi(versions[1])
					if err != nil {
						return err
					}
					t, err := tenant.Create(tid, region, uint16(mav), uint16(miv))
					uniqueFiles = append(uniqueFiles, t)
				}
			}
		}
		return nil
	}

	// Start the walk at depth 1
	if err := walk(root, 1); err != nil {
		return nil, err
	}

	return uniqueFiles, nil
}
