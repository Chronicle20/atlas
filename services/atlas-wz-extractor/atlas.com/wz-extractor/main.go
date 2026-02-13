package main

import (
	"atlas-wz-extractor/extraction"
	"atlas-wz-extractor/logger"
	"atlas-wz-extractor/service"
	"atlas-wz-extractor/tracing"
	"os"

	"github.com/Chronicle20/atlas-rest/server"
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

	p := extraction.NewProcessor(inputDir, outputXmlDir, outputImgDir)

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath("/api/").
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(extraction.InitResource(p, tdm.WaitGroup())(GetServer())).
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
