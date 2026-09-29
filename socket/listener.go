package socket

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"time"
)

type Listener struct {
	UnixListener *net.UnixListener

	Config

	stats counters
}

func (l *Listener) Consume(ctx context.Context) error {
	defer l.UnixListener.Close()

	for ctx.Err() == nil {
		c, err := l.accept()
		switch {
		case err == nil:
			l.stats.connections.Add(1)
			l.serve(ctx, c)
		case errors.Is(err, os.ErrDeadlineExceeded):
		case errors.Is(err, net.ErrClosed):
			return nil
		default:
			l.stats.acceptErrors.Add(1)
			l.onError(err)
			select {
			case <-ctx.Done():
			case <-time.After(l.AcceptTimeout):
			}
		}
	}
	return nil
}

func (l *Listener) accept() (net.Conn, error) {
	if err := l.UnixListener.SetDeadline(time.Now().Add(l.AcceptTimeout)); err != nil {
		return nil, err
	}
	return l.UnixListener.Accept()
}

func (l *Listener) serve(ctx context.Context, c net.Conn) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	// closing the conn unblocks Scan on shutdown
	go func() {
		<-ctx.Done()
		c.Close()
	}()

	scanner := bufio.NewScanner(c)
	// +1 so the newline does not count against MaxLineSize
	maxToken := l.MaxLineSize + 1
	scanner.Buffer(make([]byte, 0, min(initialBufSize, maxToken)), maxToken)

	for ctx.Err() == nil && scanner.Scan() {
		l.stats.lines.Add(1)
		l.stats.bytes.Add(uint64(len(scanner.Bytes())))
		l.Handler(scanner.Bytes())
	}
	if err := scanner.Err(); err != nil && ctx.Err() == nil {
		l.stats.readErrors.Add(1)
		l.onError(err)
	}
}

// Copy wraps h so it receives its own copy of each line, safe to keep after h returns.
func Copy(h func([]byte)) func([]byte) {
	return func(b []byte) { h(bytes.Clone(b)) }
}

func (l *Listener) onError(err error) {
	if l.OnError != nil {
		l.OnError(err)
	}
}

func NewListener(c Config) (*Listener, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	if c.MaxLineSize <= 0 {
		c.MaxLineSize = defaultMaxLineSize
	}
	if c.AcceptTimeout <= 0 {
		c.AcceptTimeout = defaultAcceptTimeout
	}

	if fi, err := os.Lstat(c.Path); err == nil && c.Force {
		if fi.Mode()&fs.ModeSocket == 0 {
			return nil, fmt.Errorf("refusing to remove %s: not a socket", c.Path)
		}
		if err := os.Remove(c.Path); err != nil {
			return nil, fmt.Errorf("unable to remove %s: %w", c.Path, err)
		}
	}

	uxl, err := net.ListenUnix("unix", &net.UnixAddr{Name: c.Path, Net: "unix"})
	if err != nil {
		return nil, &CreateError{
			Path: c.Path,
			Err:  err,
		}
	}
	return &Listener{Config: c, UnixListener: uxl}, nil
}
