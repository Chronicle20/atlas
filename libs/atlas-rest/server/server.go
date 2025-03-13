package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type ConfigFunc func(config *Config)

type Config struct {
	readTimeout  time.Duration
	writeTimeout time.Duration
	idleTimeout  time.Duration
	addr         string
}

// NewServer deprecated
//
//goland:noinspection GoUnusedExportedFunction
func NewServer(cl *logrus.Logger, ctx context.Context, wg *sync.WaitGroup, routerProducer func(l logrus.FieldLogger) http.Handler, configurators ...ConfigFunc) {
	config := &Config{
		readTimeout:  time.Duration(5) * time.Second,
		writeTimeout: time.Duration(10) * time.Second,
		idleTimeout:  time.Duration(120) * time.Second,
		addr:         ":8080",
	}

	for _, configurator := range configurators {
		configurator(config)
	}

	New(cl, ctx, wg).
		SetRouterProducer(routerProducer).
		SetReadTimeout(config.readTimeout).
		SetWriteTimeout(config.writeTimeout).
		SetIdleTimeout(config.idleTimeout).
		SetAddr(config.addr).
		Run()
}

func CommonHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

func LoggingMiddleware(l logrus.FieldLogger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			l.Debugf("Handling [%s] request on [%s]", r.Method, r.RequestURI)
			next.ServeHTTP(w, r)
		})
	}
}

type RouteProducer func(l logrus.FieldLogger) http.Handler
type Builder struct {
	l                 logrus.FieldLogger
	ctx               context.Context
	wg                *sync.WaitGroup
	w                 *io.PipeWriter
	readTimeout       time.Duration
	writeTimeout      time.Duration
	idleTimeout       time.Duration
	host              string
	port              string
	basePath          string
	routeInitializers []RouteInitializer
	routerProducer    RouteProducer
}

func New(l *logrus.Logger, ctx context.Context, wg *sync.WaitGroup) *Builder {
	sb := &Builder{}
	sb.l = l.WithFields(logrus.Fields{"originator": "HTTPServer"})
	sb.ctx = ctx
	sb.wg = wg
	sb.w = l.Writer()
	sb.readTimeout = time.Duration(5) * time.Second
	sb.writeTimeout = time.Duration(10) * time.Second
	sb.idleTimeout = time.Duration(120) * time.Second
	sb.host = ""
	sb.port = "8080"
	sb.basePath = "/"
	sb.routeInitializers = make([]RouteInitializer, 0)
	sb.routerProducer = func(l logrus.FieldLogger) http.Handler {
		return ProduceRoutes(sb.basePath, sb.routeInitializers...)(l)
	}
	return sb
}

func (sb *Builder) SetReadTimeout(t time.Duration) *Builder {
	sb.readTimeout = t
	return sb
}

func (sb *Builder) SetWriteTimeout(t time.Duration) *Builder {
	sb.writeTimeout = t
	return sb
}

func (sb *Builder) SetIdleTimeout(t time.Duration) *Builder {
	sb.idleTimeout = t
	return sb
}

func (sb *Builder) SetAddr(addr string) *Builder {
	bits := strings.Split(addr, ":")
	sb.SetHost(bits[0])
	sb.SetPort(bits[1])
	return sb
}

func (sb *Builder) SetHost(host string) *Builder {
	sb.host = host
	return sb
}

func (sb *Builder) SetPort(port string) *Builder {
	sb.port = port
	return sb
}

func (sb *Builder) SetBasePath(path string) *Builder {
	sb.basePath = path
	return sb
}

func (sb *Builder) AddRouteInitializer(initializer RouteInitializer) *Builder {
	sb.routeInitializers = append(sb.routeInitializers, initializer)
	return sb
}

func (sb *Builder) SetRouteInitializers(initializers ...RouteInitializer) *Builder {
	sb.routeInitializers = append(sb.routeInitializers, initializers...)
	return sb
}

func (sb *Builder) SetRouterProducer(producer RouteProducer) *Builder {
	sb.routerProducer = producer
	return sb
}

func (sb *Builder) Run() {
	go func() {
		hs := http.Server{
			Addr:         fmt.Sprintf("%s:%s", sb.host, sb.port),
			Handler:      sb.routerProducer(sb.l),
			ErrorLog:     log.New(sb.w, "", 0),
			ReadTimeout:  sb.readTimeout,
			WriteTimeout: sb.writeTimeout,
			IdleTimeout:  sb.idleTimeout,
		}

		sb.l.Infof("Starting server [%s:%s]", sb.host, sb.port)

		ctx, cancel := context.WithCancel(sb.ctx)
		defer cancel()

		go func() {
			sb.wg.Add(1)
			defer sb.wg.Done()
			err := hs.ListenAndServe()
			if !errors.Is(err, http.ErrServerClosed) {
				sb.l.WithError(err).Errorf("Error while serving.")
				return
			}
		}()

		<-ctx.Done()
		sb.l.Infof("Shutting down server [%s:%s]", sb.host, sb.port)
		err := hs.Close()
		if err != nil {
			sb.l.WithError(err).Errorf("Error shutting down HTTP service.")
		}
		err = sb.w.Close()
		if err != nil {
			sb.l.WithError(err).Errorf("Closing log writer.")
		}
	}()
}
