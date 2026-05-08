package main

import (
	"atlas-wz-extractor/characterimage"
	"atlas-wz-extractor/characterrender"
	"atlas-wz-extractor/extraction"
	"atlas-wz-extractor/extraction/job"
	"atlas-wz-extractor/extraction/lock"
	kproducer "atlas-wz-extractor/kafka/producer"
	"atlas-wz-extractor/logger"
	"os"
	"time"

	service "github.com/Chronicle20/atlas/libs/atlas-service"
	tracing "github.com/Chronicle20/atlas/libs/atlas-tracing"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	goredis "github.com/redis/go-redis/v9"
)

const serviceName = "atlas-wz-extractor"

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

	// Redis client for job store and tenant lock.
	redisAddr := os.Getenv("REDIS_URL")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	rc := goredis.NewClient(&goredis.Options{
		Addr:     redisAddr,
		Password: os.Getenv("REDIS_PASSWORD"),
	})

	jobStore := job.NewStore(rc)
	tenantLock := lock.NewTenantLock(rc, 30*time.Minute)

	p := extraction.NewProcessor(inputDir, outputXmlDir, outputImgDir)
	cren := characterimage.NewCompositor()
	prod := kproducer.ProviderImpl(l)(tdm.Context())
	dirs := extraction.Dirs{InputDir: inputDir, OutputXmlDir: outputXmlDir, OutputImgDir: outputImgDir}

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath("/api/").
		SetPort(os.Getenv("REST_PORT")).
		SetReadTimeout(60 * time.Minute).
		SetWriteTimeout(60 * time.Minute).
		AddRouteInitializer(extraction.InitResource(p, jobStore, tenantLock, prod, tdm.WaitGroup(), dirs)(GetServer())).
		AddRouteInitializer(characterrender.InitResource(outputImgDir, cren)(GetServer())).
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
