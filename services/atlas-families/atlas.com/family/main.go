package main

import (
	"atlas-family/database"
	"atlas-family/family"
	"atlas-family/kafka/consumer"
	familyconsumer "atlas-family/kafka/consumer/family"
	"atlas-family/kafka/producer"
	"atlas-family/logger"
	"atlas-family/scheduler"
	"atlas-family/service"
	"atlas-family/tracing"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"os"
)

const serviceName = "atlas-family"

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

	tc, err := tracing.InitTracer(l)(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	// Initialize database connection
	db := database.Connect(l)
	if db == nil {
		l.Fatal("Failed to connect to database")
	}

	// Run database migrations
	if err := family.Migration(db); err != nil {
		l.WithError(err).Fatal("Failed to run database migrations")
	}

	// Initialize Kafka producer
	kafkaProducer := producer.ProviderImpl(l)(tdm.Context())

	// Initialize family components
	administrator := family.NewAdministrator(db, l)
	processor := family.NewProcessor(db, l, administrator, kafkaProducer)

	// Initialize and start Kafka consumers
	consumerGroupId := serviceName + "-consumer"
	familyconsumer.InitConsumers(l)(func(config consumer.Config, decorators ...func(consumer.Config) consumer.Config) {
		c := config
		for _, decorator := range decorators {
			c = decorator(c)
		}
		go consumer.Start(l, c, processor)
	})(consumerGroupId)

	// Initialize and start reputation reset scheduler
	reputationResetJob := scheduler.NewReputationResetJobFromEnv(l, administrator, kafkaProducer)
	if err := reputationResetJob.Start(tdm.Context()); err != nil {
		l.WithError(err).Fatal("Failed to start reputation reset job")
	}

	// Setup graceful shutdown for scheduler
	tdm.TeardownFunc(func() {
		reputationResetJob.Stop()
	})

	// Setup route handler with family endpoints
	routeHandler := func(router *mux.Router) {
		family.RegisterRoutes(router, l, db, processor, administrator)
	}

	// CreateRoute and run server
	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		SetPort(os.Getenv("REST_PORT")).
		WithRouteHandler(routeHandler).
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))

	tdm.Wait()
	l.Infoln("Service shutdown.")
}
