package main

import (
	"atlas-dis/continent"
	drop2 "atlas-dis/continent/drop"
	"atlas-dis/database"
	"atlas-dis/logger"
	"atlas-dis/monster"
	"atlas-dis/monster/drop"
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

	db := database.Connect(l, database.SetMigrations(drop.Migration, drop2.Migration))

	rest.CreateService(l, db, ctx, wg, GetServer().GetPrefix(), drop.InitResource(GetServer()), monster.InitResource(GetServer()), continent.InitResource(GetServer()))

	initializeMonsterDrops(l, db)
	initializeContinentDrops(l, db)

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

func initializeMonsterDrops(l logrus.FieldLogger, db *gorm.DB) {
	s := drop.GetAll(l, db)
	if len(s) > 0 {
		return
	}

	filePath, ok := os.LookupEnv("MONSTER_JSON_FILE_PATH")
	if !ok {
		l.Fatalf("Environment variable MONSTER_JSON_FILE_PATH is not set.")
	}

	jsonData, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Fatal("Error reading JSON file:", err)
	}

	// Define a slice to store the objects
	var objects []drop.JSONModel
	var drops []drop.Model

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
			SetQuestId(jdo.QuestId).
			SetChance(jdo.Chance).
			Build()
		drops = append(drops, md)
	}

	err = drop.BulkCreateMonsterDrop(db, drops)
	if err != nil {
		l.Fatalf(err.Error())
	}
}

func initializeContinentDrops(l logrus.FieldLogger, db *gorm.DB) {
	s := drop2.GetAll(l, db)
	if len(s) > 0 {
		return
	}

	filePath, ok := os.LookupEnv("CONTINENT_JSON_FILE_PATH")
	if !ok {
		l.Fatalf("Environment variable CONTINENT_JSON_FILE_PATH is not set.")
	}

	jsonData, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Fatal("Error reading JSON file:", err)
	}

	// Define a slice to store the objects
	var objects []drop2.JSONModel
	var drops []drop2.Model

	// Unmarshal JSON into the slice
	err = json.Unmarshal(jsonData, &objects)
	if err != nil {
		log.Fatal("Error unmarshalling JSON:", err)
	}

	for _, jdo := range objects {
		md := drop2.NewContinentDropBuilder(0).
			SetContinentId(jdo.ContinentId).
			SetItemId(jdo.ItemId).
			SetMinimumQuantity(jdo.MinimumQuantity).
			SetMaximumQuantity(jdo.MaximumQuantity).
			SetQuestId(jdo.QuestId).
			SetChance(jdo.Chance).
			Build()
		drops = append(drops, md)
	}

	err = drop2.BulkCreateContinentDrop(db, drops)
	if err != nil {
		l.Fatalf(err.Error())
	}
}
