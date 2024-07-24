package main

import (
	"atlas-channel/channel"
	"atlas-channel/configuration"
	"atlas-channel/kafka/consumer/character"
	"atlas-channel/kafka/consumer/map"
	"atlas-channel/logger"
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
	"github.com/Chronicle20/atlas-socket/response"
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
	handlerMap[handler.PortalScriptHandle] = handler.PortalScriptHandleFunc
	handlerMap[handler.MapChangeHandle] = handler.MapChangeHandleFunc

	writerList := []string{
		writer.SetField,
		writer.SpawnNPC,
		writer.SpawnNPCRequestController,
		writer.NPCAction,
		writer.StatChanged,
	}

	cm := consumer.GetManager()
	cm.AddConsumer(l, ctx, wg)(_map.StatusEventConsumer(l)(fmt.Sprintf(consumerGroupId, config.Data.Id)))
	cm.AddConsumer(l, ctx, wg)(character.StatusEventConsumer(l)(fmt.Sprintf(consumerGroupId, config.Data.Id)))
	cm.AddConsumer(l, ctx, wg)(channel.CommandStatusConsumer(l)(fmt.Sprintf(consumerGroupId, config.Data.Id)))

	for _, s := range config.Data.Attributes.Servers {
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

				// TODO this needs refactoring.
				var wp writer.Producer
				if sc.Tenant().Region == "GMS" && sc.Tenant().MajorVersion <= 28 {
					owp := func(op uint8) writer.OpWriter {
						return func(w *response.Writer) {
							w.WriteByte(op)
						}
					}
					wp = getWriterProducer[uint8](l)(s.Writers, writerList, owp)
				} else {
					owp := func(op uint16) writer.OpWriter {
						return func(w *response.Writer) {
							w.WriteShort(op)
						}
					}
					wp = getWriterProducer[uint16](l)(s.Writers, writerList, owp)
				}
				_, _ = cm.RegisterHandler(_map.StatusEventCharacterEnterRegister(sc, wp)(l))
				_, _ = cm.RegisterHandler(_map.StatusEventCharacterExitRegister(sc, wp)(l))
				_, _ = cm.RegisterHandler(character.StatusEventStatChangedRegister(sc, wp)(l))
				_, _ = cm.RegisterHandler(character.StatusEventMapChangedRegister(sc, wp)(l))
				_, _ = cm.RegisterHandler(channel.CommandStatusRegister(sc, config.Data.Attributes.IPAddress, c.Port)(l))

				socket.CreateSocketService(l, ctx, wg)(s, validatorMap, handlerMap, writerList, sc, config.Data.Attributes.IPAddress, c.Port)
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

func getWriterProducer[E uint8 | uint16](l logrus.FieldLogger) func(writerConfig []configuration.Writer, wl []string, opwp writer.OpWriterProducer[E]) writer.Producer {
	return func(writerConfig []configuration.Writer, wl []string, opwp writer.OpWriterProducer[E]) writer.Producer {
		rwm := make(map[string]writer.BodyFunc)
		for _, wc := range writerConfig {
			op, err := strconv.ParseUint(wc.OpCode, 0, 16)
			if err != nil {
				l.WithError(err).Errorf("Unable to configure writer [%s] for opcode [%s].", wc.Writer, wc.OpCode)
				continue
			}

			for _, wn := range wl {
				if wn == wc.Writer {
					rwm[wc.Writer] = writer.MessageGetter(opwp(E(op)), wc.Options)
				}
			}
		}
		return writer.ProducerGetter(rwm)
	}
}
