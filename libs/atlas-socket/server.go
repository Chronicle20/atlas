package socket

import (
	"context"
	"errors"
	"fmt"
	"github.com/Chronicle20/atlas-socket/crypto"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"io"
	"net"
	"os"
	"sync"
	"time"
)

type OpReader interface {
	Read(r *request.Reader) uint16
}

type OpWriter interface {
	Write(op uint16) func(w *response.Writer)
}

type OpReadWriter interface {
	OpReader
	OpWriter
}

type ByteReadWriter struct {
}

func (b ByteReadWriter) Read(r *request.Reader) uint16 {
	return uint16(r.ReadByte())
}

func (b ByteReadWriter) Write(op uint16) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteByte(byte(op))
	}
}

type ShortReadWriter struct {
}

func (s ShortReadWriter) Read(r *request.Reader) uint16 {
	return r.ReadUint16()
}

func (s ShortReadWriter) Write(op uint16) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteShort(op)
	}
}

type HandlerProducer func() map[uint16]request.Handler

type Creator func(sessionId uuid.UUID, conn net.Conn)

func defaultCreator(_ uuid.UUID, _ net.Conn) {
}

type MessageDecryptor func(sessionId uuid.UUID, message []byte) []byte

func defaultMessageDecryptor(_ uuid.UUID, message []byte) []byte {
	return message
}

type Destroyer func(sessionId uuid.UUID)

func defaultDestroyer(_ uuid.UUID) {
}

type config struct {
	rw        OpReadWriter
	creator   Creator
	decryptor MessageDecryptor
	destroyer Destroyer
	ipAddress string
	port      int
	handlers  map[uint16]request.Handler
}

//goland:noinspection GoUnusedExportedFunction
func Run(l logrus.FieldLogger, ctx context.Context, wg *sync.WaitGroup, configurators ...Configurator) error {
	wg.Add(1)
	defer wg.Done()

	c := &config{
		creator:   defaultCreator,
		decryptor: defaultMessageDecryptor,
		destroyer: defaultDestroyer,
		ipAddress: "0.0.0.0",
		port:      5000,
		handlers:  make(map[uint16]request.Handler),
	}

	for _, configurator := range configurators {
		configurator(c)
	}

	l.Infof("Starting tcp server on [%s:%d]", c.ipAddress, c.port)
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", c.ipAddress, c.port))
	if err != nil {
		l.WithError(err).Errorln("Error listening:", err.Error())
		return err
	}

	defer func(lis net.Listener) {
		err := lis.Close()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			l.WithError(err).Error("Error closing listener")
		}
	}(lis)

	go func() {
		<-ctx.Done()
		l.Infof("Closing listener.")
		err := lis.Close()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			l.WithError(err).Errorf("Error closing listener.")
		}
	}()

	for {
		conn, err := lis.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				l.Infof("Listener stopped accepting new connections.")
				return err
			default:
				l.WithError(err).Infof("Error accepting connection.")
				continue
			}
		}

		l.Infof("Client [%s] connected.", conn.RemoteAddr())

		go run(l, ctx, wg)(c, conn, uuid.New(), 4)
	}
}

func run(l logrus.FieldLogger, ctx context.Context, wg *sync.WaitGroup) func(config *config, conn net.Conn, sessionId uuid.UUID, headerSize int) {
	return func(config *config, conn net.Conn, sessionId uuid.UUID, headerSize int) {
		wg.Add(1)
		defer wg.Done()

		defer func(conn net.Conn) {
			err := conn.Close()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}
				l.WithError(err).Errorf("Error closing connection.")
			} else {
				l.Infof("Closing connection from [%s].", conn.RemoteAddr())
			}
		}(conn)

		go func() {
			<-ctx.Done()
			l.Infof("Closing connection from [%s].", conn.RemoteAddr())
			conn.Close()
		}()

		config.creator(sessionId, conn)

		header := true
		readSize := headerSize

		fl := l.WithField("session", sessionId.String())

		for {
			buffer := make([]byte, readSize)

			_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			_, err := conn.Read(buffer)
			if err != nil {
				if os.IsTimeout(err) {
					continue
				}
				if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
					l.Infof("Connection ended.")
				} else {
					l.WithError(err).Errorf("Error reading from connection.")
				}
				config.destroyer(sessionId)
				return
			}

			if header {
				readSize = crypto.PacketLength(buffer)
			} else {
				readSize = headerSize

				result := buffer
				result = config.decryptor(sessionId, buffer)
				go handle(fl)(config, sessionId, result)
			}

			header = !header
		}
	}
}

func handle(l logrus.FieldLogger) func(config *config, sessionId uuid.UUID, p request.Request) {
	return func(config *config, sessionId uuid.UUID, p request.Request) {
		reader := request.NewRequestReader(&p, time.Now().Unix())
		op := config.rw.Read(&reader)
		if h, ok := config.handlers[op]; ok {
			h(sessionId, reader)
		} else {
			l.Infof("Read a unhandled message with op 0x%02X.", op&0xFF)
		}
	}
}
