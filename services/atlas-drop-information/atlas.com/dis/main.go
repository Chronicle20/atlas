package main

import (
	"atlas-dis/database"
	"atlas-dis/drop"
	"atlas-dis/logger"
	"atlas-dis/rest"
	"atlas-dis/tracing"
	"context"
	"encoding/json"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)
import "gorm.io/gorm"

const serviceName = "atlas-dis"

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
		prefix:  "/api/dis/",
	}
}

func main() {
	l := logger.CreateLogger(serviceName)
	l.Infoln("Starting main service.")

	wg := &sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())

	tc, err := tracing.InitTracer(l)(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}
	defer func(tc io.Closer) {
		err := tc.Close()
		if err != nil {
			l.WithError(err).Errorf("Unable to close tracer.")
		}
	}(tc)

	db := database.Connect(l, database.SetMigrations(drop.Migration))

	rest.CreateService(l, db, ctx, wg, GetServer().GetPrefix(), drop.InitResource(GetServer()))

	initializeDrops(l, db)

	// trap sigterm or interrupt and gracefully shutdown the server
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill, syscall.SIGTERM)

	// Block until a signal is received.
	sig := <-c
	l.Infof("Initiating shutdown with signal %s.", sig)
	cancel()
	wg.Wait()
	l.Infoln("Service shutdown.")
}

func initializeDrops(l logrus.FieldLogger, db *gorm.DB) {
	s, err := drop.GetAll(l, db)
	if err != nil {
		l.Fatalf(err.Error())
	}
	if len(s) > 0 {
		return
	}

	filePath, ok := os.LookupEnv("JSON_FILE_PATH")
	if !ok {
		l.Fatalf("Environment variable JSON_FILE_PATH is not set.")
	}

	jsonData, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Fatal("Error reading JSON file:", err)
	}

	// Define a slice to store the objects
	var objects []drop.JSONModel
	var monsterDrops []drop.Model

	// Unmarshal JSON into the slice
	err = json.Unmarshal(jsonData, &objects)
	if err != nil {
		log.Fatal("Error unmarshalling JSON:", err)
	}

	for _, jdo := range objects {
		md := drop.NewMonsterDropBuilder(0).
			SetMonsterId(jdo.MonsterId).
			SetItemId(jdo.ItemId).
			SetMinimumQuantity(jdo.MinimumQuantity).
			SetMaximumQuantity(jdo.MaximumQuantity).
			SetChance(jdo.Chance).
			Build()
		monsterDrops = append(monsterDrops, md)
	}

	err = drop.BulkCreateMonsterDrop(db, monsterDrops)
	if err != nil {
		l.Fatalf(err.Error())
	}
}
