package socket

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	"github.com/Chronicle20/atlas/libs/atlas-socket/crypto"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const (
	// readTimeout bounds a single Read so the loop can periodically run idle
	// detection instead of blocking indefinitely.
	readTimeout = 5 * time.Second
	// partialFrameTimeout bounds how long a half-delivered frame may stall
	// before the session is torn down. Only reached mid-frame, where waiting is
	// correct but unbounded waiting is not.
	partialFrameTimeout = 30 * time.Second
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

type ByteReadWriter struct{}

func (b ByteReadWriter) Read(r *request.Reader) uint16 {
	return uint16(r.ReadByte())
}

func (b ByteReadWriter) Write(op uint16) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteByte(byte(op))
	}
}

type ShortReadWriter struct{}

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
	rw            OpReadWriter
	creator       Creator
	decryptor     MessageDecryptor
	destroyer     Destroyer
	ipAddress     string
	port          int
	handlers      map[uint16]request.Handler
	idleNotifier  IdleNotifier
	idleThreshold time.Duration
}

// WaitGrouper is the minimal surface Serve/Run needs from a waitgroup. A
// *sync.WaitGroup satisfies it with no caller changes; atlas-channel uses
// it to fan Add/Done out to more than one underlying waitgroup
// (task-244 design.md §3.1).
type WaitGrouper interface {
	Add(delta int)
	Done()
}

func buildConfig(configurators ...Configurator) *config {
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
	return c
}

// Bind performs net.Listen synchronously. Callers that must observe a
// bind failure before starting any side effect -- listener.Registry.Add's
// rollback path in atlas-channel -- call this instead of Run.
func Bind(l logrus.FieldLogger, ipAddress string, port int) (net.Listener, error) {
	l.Infof("Starting tcp server on [%s:%d]", ipAddress, port)
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", ipAddress, port))
	if err != nil {
		l.WithError(err).Errorf("Error listening on [%s:%d].", ipAddress, port)
		return nil, err
	}
	return lis, nil
}

// Serve runs the accept loop against an already-bound listener. wg
// brackets Serve's own lifetime (Add(1) on entry, Done() on return),
// matching what Run does today. sessionWg is incremented once per
// accepted connection AT THE ACCEPT SITE and released by the
// per-connection goroutine, so a caller can track session lifetime
// independently of the listener's. The accept-site increment is what
// closes the accept->Add gap and makes a concurrent Wait safe
// (task-244 design.md §2.3).
//
// Serve ignores the ipAddress/port configurators -- the listener is
// already bound. Every other configurator still applies.
func Serve(l logrus.FieldLogger, ctx context.Context, wg WaitGrouper, sessionWg WaitGrouper, lis net.Listener, configurators ...Configurator) error {
	wg.Add(1)
	defer wg.Done()

	c := buildConfig(configurators...)

	defer func(lis net.Listener) {
		err := lis.Close()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			l.WithError(err).Error("Error closing listener")
		}
	}(lis)

	routine.Go(l, ctx, func(_ context.Context) {
		<-ctx.Done()
		l.Infof("Closing listener.")
		err := lis.Close()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			l.WithError(err).Errorf("Error closing listener.")
		}
	})

	for {
		conn, err := lis.Accept()
		if err != nil {
			// A closed listener is terminal whether or not ctx has been
			// canceled -- drain phase 1 closes the listener while the
			// handle's ctx is still live, and a ctx-only check would spin
			// a hot loop here (task-244 design.md §3.3).
			if errors.Is(err, net.ErrClosed) {
				l.Infof("Listener stopped accepting new connections.")
				return err
			}
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

		// Add before the goroutine starts: run()'s own Add(1) would race a
		// concurrent Wait and panic with "WaitGroup misuse".
		sessionWg.Add(1)
		routine.Go(l, ctx, func(_ context.Context) { run(l, ctx, sessionWg)(c, conn, uuid.New(), 4) })
	}
}

//goland:noinspection GoUnusedExportedFunction
func Run(l logrus.FieldLogger, ctx context.Context, wg WaitGrouper, configurators ...Configurator) error {
	c := buildConfig(configurators...)
	lis, err := Bind(l, c.ipAddress, c.port)
	if err != nil {
		return err
	}
	return Serve(l, ctx, wg, wg, lis, configurators...)
}

func run(l logrus.FieldLogger, ctx context.Context, wg WaitGrouper) func(config *config, conn net.Conn, sessionId uuid.UUID, headerSize int) {
	return func(config *config, conn net.Conn, sessionId uuid.UUID, headerSize int) {
		// The matching Add(1) happens at Serve's accept site, before this
		// goroutine is spawned (task-244 design.md §2.3).
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

		routine.Go(l, ctx, func(_ context.Context) {
			<-ctx.Done()
			l.Infof("Closing connection from [%s].", conn.RemoteAddr())
			conn.Close()
		})

		config.creator(sessionId, conn)

		header := true
		readSize := headerSize

		fl := l.WithField("session", sessionId.String())

		// Idle state tracking
		lastActivity := time.Now()
		isIdle := false

		for {
			buffer := make([]byte, readSize)

			// net.Conn.Read returns whatever has already arrived, up to len(buffer) --
			// it does not fill the buffer. Bodies that exceed one TCP segment (~1460
			// bytes on a typical path) therefore arrive across several reads, so read
			// until the frame is complete. Taking a short read as a whole frame both
			// hands the decryptor a zero-padded tail (AES-OFB turns those zeros into
			// raw keystream, i.e. plausible-looking garbage) and leaves the remainder
			// in the socket, where the next iteration consumes it as a header --
			// desynchronising the stream permanently. Small packets hid this because
			// they almost always arrive in one segment.
			read := 0
			stalledSince := time.Now()
			for read < len(buffer) {
				_ = conn.SetReadDeadline(time.Now().Add(readTimeout))
				n, err := conn.Read(buffer[read:])
				read += n
				if n > 0 {
					stalledSince = time.Now()
				}
				if err == nil {
					continue
				}

				if os.IsTimeout(err) {
					// Idle detection only applies at a frame boundary. Mid-frame the peer
					// still owes us bytes, so a quiet socket is an incomplete send rather
					// than an inactive client -- keep waiting, but not forever: a peer that
					// announces a length and then stops would otherwise pin this goroutine.
					if read == 0 && header {
						if config.idleNotifier != nil && !isIdle && config.idleThreshold > 0 {
							if time.Since(lastActivity) > config.idleThreshold {
								isIdle = true
								config.idleNotifier(sessionId)
							}
						}
						continue
					}
					if time.Since(stalledSince) < partialFrameTimeout {
						continue
					}
					l.Warnf("Abandoning partial frame: got [%d] of [%d] bytes before the peer went quiet for [%s].", read, len(buffer), partialFrameTimeout)
					config.destroyer(sessionId)
					return
				}

				if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) || errors.Is(err, syscall.ECONNRESET) {
					l.Infof("Connection ended.")
				} else {
					l.WithError(err).Errorf("Error reading from connection.")
				}
				config.destroyer(sessionId)
				return
			}

			// Data received - reset activity tracking
			lastActivity = time.Now()
			isIdle = false

			if header {
				readSize = crypto.PacketLength(buffer)
			} else {
				readSize = headerSize

				result := buffer
				result = config.decryptor(sessionId, buffer)
				routine.Go(fl, ctx, func(_ context.Context) { handle(fl)(config, sessionId, result) })
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
