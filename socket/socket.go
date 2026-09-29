package socket

import (
	"errors"
	"fmt"
	"time"
)

const (
	defaultMaxLineSize   = 32 * 1024 * 1024
	initialBufSize       = 64 * 1024
	defaultAcceptTimeout = time.Second
)

type CreateError struct {
	Path string
	Err  error
}

func (e CreateError) Error() string {
	return fmt.Sprintf("socket: %s Error: [%s]", e.Path, e.Err)
}

type Config struct {
	Path  string
	Force bool
	// Handler gets the scanner buffer, overwritten on the next line; wrap with Copy to keep it.
	Handler func([]byte)
	OnError func(error)
	// MaxLineSize defaults to 32 MiB; alerts with printable payloads exceed bufio's 64 KiB.
	MaxLineSize int
	// AcceptTimeout bounds how long shutdown waits on an idle listener; defaults to 1s.
	AcceptTimeout time.Duration
}

func (c *Config) Validate() error {
	if c.Path == "" {
		return errors.New("unix socket path missing")
	}
	if c.Handler == nil {
		return errors.New("handler missing")
	}
	return nil
}
