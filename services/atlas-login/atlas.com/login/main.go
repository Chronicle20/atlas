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
	"github.com/opentracing/opentracing-go"
	"io"
	"os"
	"os/signal"
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
	handlerMap[handler.DebugHandle] = handler.DebugHandleFunc
	handlerMap[handler.CreateSecurityHandle] = handler.CreateSecurityHandleFunc
	handlerMap[handler.LoginHandle] = handler.LoginHandleFunc
	handlerMap[handler.ServerListRequestHandle] = handler.ServerListRequestHandleFunc
	handlerMap[handler.ServerStatusHandle] = handler.ServerStatusHandleFunc
	handlerMap[handler.CharacterListWorldHandle] = handler.CharacterListWorldHandleFunc
	handlerMap[handler.CharacterCheckNameHandle] = handler.CharacterCheckNameHandleFunc
	handlerMap[handler.CreateCharacterHandle] = handler.CreateCharacterHandleFunc
	handlerMap[handler.DeleteCharacterHandle] = handler.DeleteCharacterHandleFunc
	handlerMap[handler.AfterLoginHandle] = handler.AfterLoginHandleFunc
	handlerMap[handler.RegisterPinHandle] = handler.RegisterPinHandleFunc
	handlerMap[handler.RegisterPicHandle] = handler.RegisterPicHandleFunc
	handlerMap[handler.AcceptTosHandle] = handler.AcceptTosHandleFunc
	handlerMap[handler.CharacterSelectedHandle] = handler.CharacterSelectedHandleFunc
	handlerMap[handler.CharacterSelectedPicHandle] = handler.CharacterSelectedPicHandleFunc
	handlerMap[handler.WorldSelectHandle] = handler.WorldSelectHandleFunc
	handlerMap[handler.SetGenderHandle] = handler.SetGenderHandleFunc

	writerList := []string{
		writer.LoginAuth,
		writer.AuthSuccess,
		writer.AuthTemporaryBan,
		writer.AuthPermanentBan,
		writer.AuthLoginFailed,
		writer.ServerListRecommendations,
		writer.ServerListEntry,
		writer.ServerListEnd,
		writer.SelectWorld,
		writer.ServerStatus,
		writer.CharacterList,
		writer.CharacterNameResponse,
		writer.AddCharacterEntry,
		writer.DeleteCharacterResponse,
		writer.PinOperation,
		writer.PinUpdate,
		writer.PicResult,
		writer.ServerIP,
		writer.ServerLoad,
		writer.SetAccountResult,
	}

	for _, s := range config.Data.Attributes.Servers {
		socket.CreateSocketService(l, ctx, wg)(s, validatorMap, handlerMap, writerList)
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
	session.DestroyAll(l, span, session.GetRegistry())

	l.Infoln("Service shutdown.")
}
