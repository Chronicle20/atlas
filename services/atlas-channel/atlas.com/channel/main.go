package main

import (
	"atlas-channel/configuration"
	"atlas-channel/logger"
	_map "atlas-channel/map"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket"
	"atlas-channel/socket/handler"
	"atlas-channel/socket/writer"
	"atlas-channel/tasks"
	"atlas-channel/tenant"
	"atlas-channel/tracing"
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
)

const serviceName = "atlas-channel"
const consumerGroupId = "Channel Service - %s"

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
		prefix:  "/api/channel/",
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

	config, err := configuration.GetConfiguration()
	if err != nil {
		l.WithError(err).Fatal("Unable to successfully load configuration.")
	}

	validatorMap := make(map[string]handler.MessageValidator)
	validatorMap[handler.NoOpValidator] = handler.NoOpValidatorFunc
	validatorMap[handler.LoggedInValidator] = handler.LoggedInValidatorFunc

	handlerMap := make(map[string]handler.MessageHandler)
	handlerMap[handler.NoOpHandler] = handler.NoOpHandlerFunc
	handlerMap[handler.CharacterLoggedInHandle] = handler.CharacterLoggedInHandleFunc
	handlerMap[handler.NPCActionHandle] = handler.NPCActionHandleFunc

	writerMap := make(map[string]writer.HeaderFunc)
	writerMap[writer.SetField] = writer.MessageGetter
	writerMap[writer.SpawnNPC] = writer.MessageGetter
	writerMap[writer.SpawnNPCRequestController] = writer.MessageGetter
	writerMap[writer.NPCAction] = writer.MessageGetter

	cm := consumer.GetManager()
	cm.AddConsumer(l, ctx, wg)(_map.StatusEventConsumer(l)(fmt.Sprintf(consumerGroupId, uuid.New().String())))

	for _, s := range config.Data.Attributes.Servers {
		wp := getWriterProducer(l)(s.Writers, writerMap)

		for _, w := range s.Worlds {
			for _, c := range w.Channels {
				var t tenant.Model
				t, err = tenant.NewFromConfiguration(l)(s)
				if err != nil {
					continue
				}
				var sc server.Model
				sc, err = server.New(t, w.Id, c.Id)
				if err != nil {
					continue
				}

				_, _ = cm.RegisterHandler(_map.StatusEventCharacterEnterRegister(sc, wp)(l))
				_, _ = cm.RegisterHandler(_map.StatusEventCharacterExitRegister(sc, wp)(l))

				socket.CreateSocketService(l, ctx, wg)(s, validatorMap, handlerMap, wp, sc, config.Data.Attributes.IPAddress, c.Port)
			}
		}
	}

	tt, err := config.FindTask(session.TimeoutTask)
	if err != nil {
		l.WithError(err).Fatalf("Unable to find task [%s].", session.TimeoutTask)
	}
	go tasks.Register(l, ctx)(session.NewTimeout(l, time.Millisecond*time.Duration(tt.Attributes.Interval)))

	// trap sigterm or interrupt and gracefully shutdown the server
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill, syscall.SIGTERM)

	// Block until a signal is received.
	sig := <-c
	l.Infof("Initiating shutdown with signal %s.", sig)
	cancel()
	wg.Wait()

	span := opentracing.StartSpan("teardown")
	defer span.Finish()
	tenant.ForAll(session.DestroyAll(l, span, session.GetRegistry()))

	l.Infoln("Service shutdown.")
}

func getWriterProducer(l logrus.FieldLogger) func(writerConfig []configuration.Writer, wm map[string]writer.HeaderFunc) writer.Producer {
	return func(writerConfig []configuration.Writer, wm map[string]writer.HeaderFunc) writer.Producer {
		rwm := make(map[string]writer.BodyFunc)
		for _, wc := range writerConfig {
			op, err := strconv.ParseUint(wc.OpCode, 0, 16)
			if err != nil {
				l.WithError(err).Errorf("Unable to configure writer [%s] for opcode [%s].", wc.Writer, wc.OpCode)
				continue
			}

			if w, ok := wm[wc.Writer]; ok {
				rwm[wc.Writer] = w(uint16(op), wc.Options)
			}
		}
		return writer.ProducerGetter(rwm)
	}
}
