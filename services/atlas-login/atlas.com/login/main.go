package main

import (
	"atlas-login/configuration"
	"atlas-login/logger"
	"atlas-login/session"
	"atlas-login/socket"
	"atlas-login/socket/handler"
	"atlas-login/socket/writer"
	"atlas-login/tasks"
	"atlas-login/tracing"
	"context"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
)

const serviceName = "atlas-login"

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
		prefix:  "/api/login/",
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
	handlerMap[handler.CreateSecurityHandle] = handler.CreateSecurityHandleFunc
	handlerMap[handler.LoginHandle] = handler.LoginHandleFunc
	handlerMap[handler.ServerListRequestHandle] = handler.ServerListRequestHandleFunc
	handlerMap[handler.ServerStatusHandle] = handler.ServerStatusHandleFunc
	handlerMap[handler.CharacterListWorldHandle] = handler.CharacterListWorldHandleFunc

	writerMap := make(map[string]writer.HeaderFunc)
	writerMap[writer.LoginAuth] = writer.MessageGetter
	writerMap[writer.AuthSuccess] = writer.MessageGetter
	writerMap[writer.AuthTemporaryBan] = writer.MessageGetter
	writerMap[writer.AuthPermanentBan] = writer.MessageGetter
	writerMap[writer.AuthLoginFailed] = writer.MessageGetter
	writerMap[writer.ServerListRecommendations] = writer.MessageGetter
	writerMap[writer.ServerListEntry] = writer.MessageGetter
	writerMap[writer.ServerListEnd] = writer.MessageGetter
	writerMap[writer.SelectWorld] = writer.MessageGetter
	writerMap[writer.ServerStatus] = writer.MessageGetter
	writerMap[writer.CharacterList] = writer.MessageGetter

	for _, s := range config.Data.Attributes.Servers {
		wp := getWriterProducer(l)(s.Writers, writerMap)
		socket.CreateSocketService(l, ctx, wg)(s, validatorMap, handlerMap, wp)
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
