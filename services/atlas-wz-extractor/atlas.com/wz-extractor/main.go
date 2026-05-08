package main

import (
	"atlas-wz-extractor/characterimage"
	"atlas-wz-extractor/characterrender"
	"atlas-wz-extractor/extraction"
	"atlas-wz-extractor/extraction/job"
	"atlas-wz-extractor/extraction/lock"
	extconsumer "atlas-wz-extractor/kafka/consumer/extraction"
	wzproducer "atlas-wz-extractor/kafka/producer"
	"atlas-wz-extractor/logger"
	"context"
	"os"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	atlasredis "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	service "github.com/Chronicle20/atlas/libs/atlas-service"
	tracing "github.com/Chronicle20/atlas/libs/atlas-tracing"
)

const serviceName = "atlas-wz-extractor"
const consumerGroupId = "wz-extractor-extraction"

type Server struct {
	baseUrl string
	prefix  string
}

func (s Server) GetBaseURL() string { return s.baseUrl }
func (s Server) GetPrefix() string  { return s.prefix }

func GetServer() Server { return Server{baseUrl: "", prefix: "/api/"} }

const lockTTL = 60 * time.Minute

func main() {
	l := logger.CreateLogger(serviceName)
	l.Infoln("Starting main service.")

	inputDir := os.Getenv("INPUT_WZ_DIR")
	outputXmlDir := os.Getenv("OUTPUT_XML_DIR")
	outputImgDir := os.Getenv("OUTPUT_IMG_DIR")
	if inputDir == "" || outputXmlDir == "" || outputImgDir == "" {
		l.Fatal("Required environment variables not set: INPUT_WZ_DIR, OUTPUT_XML_DIR, OUTPUT_IMG_DIR")
	}

	tdm := service.GetTeardownManager()

	tc, err := tracing.InitTracer(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	rc := atlasredis.Connect(l)
	defer rc.Close()

	store := job.NewStore(rc)
	tl := lock.NewTenantLock(rc, lockTTL)

	p := extraction.NewProcessor(inputDir, outputXmlDir, outputImgDir)
	cren := characterimage.NewCompositor()

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	extconsumer.InitConsumers(l)(cmf)(consumerGroupId)
	if err := extconsumer.InitHandlers(l)(p, store, tl)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}

	tdm.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	prodProvider := wzproducer.ProviderImpl(l)(context.Background())

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		SetReadTimeout(60 * time.Minute).
		SetWriteTimeout(60 * time.Minute).
		AddRouteInitializer(extraction.InitResource(p, store, tl, prodProvider, tdm.WaitGroup(), extraction.Dirs{InputDir: inputDir, OutputXmlDir: outputXmlDir, OutputImgDir: outputImgDir})(GetServer())).
		AddRouteInitializer(characterrender.InitResource(outputImgDir, cren)(GetServer())).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))
	tdm.Wait()
	l.Infoln("Service shutdown.")
}
