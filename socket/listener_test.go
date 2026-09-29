package socket

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// t.TempDir embeds the test name and can exceed the unix socket path limit.
func sockPath(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "gs") //nolint:usetesting // sun_path is limited to 104-108 bytes
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })
	return filepath.Join(dir, "s.sock")
}

func startListener(t *testing.T, cfg Config) (path string, stop func()) {
	t.Helper()
	cfg.Path = sockPath(t)
	cfg.AcceptTimeout = 50 * time.Millisecond

	l, err := NewListener(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- l.Consume(ctx) }()

	var once sync.Once
	stop = func() {
		once.Do(func() {
			cancel()
			select {
			case err := <-done:
				require.NoError(t, err)
			case <-time.After(3 * time.Second):
				t.Fatal("Consume did not return after cancel")
			}
		})
	}
	t.Cleanup(stop)
	return cfg.Path, stop
}

func dial(t *testing.T, path string) net.Conn {
	t.Helper()
	c, err := net.Dial("unix", path)
	require.NoError(t, err)
	t.Cleanup(func() { c.Close() })
	return c
}

func recv(t *testing.T, ch <-chan []byte, n int) []string {
	t.Helper()
	out := make([]string, 0, n)
	for range n {
		select {
		case b := <-ch:
			out = append(out, string(b))
		case <-time.After(3 * time.Second):
			t.Fatalf("got %d of %d lines", len(out), n)
		}
	}
	return out
}

func TestConsumeLines(t *testing.T) {
	lines := []string{
		`{"event_type":"alert","src_ip":"10.0.0.1","alert":{"signature_id":2100498}}`,
		`{"event_type":"flow","src_ip":"10.0.0.2","flow":{"pkts_toserver":3}}`,
		`{"event_type":"dns","dns":{"rrname":"example.com"}}`,
	}
	ch := make(chan []byte, len(lines))
	path, _ := startListener(t, Config{Handler: Copy(func(b []byte) { ch <- b })})

	c := dial(t, path)
	for _, l := range lines {
		_, err := c.Write([]byte(l + "\n"))
		require.NoError(t, err)
	}
	require.Equal(t, lines, recv(t, ch, len(lines)))
}

// Lines kept past the handler call via Copy must not be overwritten by the scanner.
func TestConsumeCopiesLines(t *testing.T) {
	const n = 1000
	want := make([]string, n)
	for i := range want {
		want[i] = fmt.Sprintf(`{"seq":%d,"pad":%q}`, i, strings.Repeat("x", 64+i%97))
	}
	ch := make(chan []byte, n)
	path, _ := startListener(t, Config{Handler: Copy(func(b []byte) { ch <- b })})

	c := dial(t, path)
	_, err := c.Write([]byte(strings.Join(want, "\n") + "\n"))
	require.NoError(t, err)
	require.Eventually(t, func() bool { return len(ch) == n }, 3*time.Second, 10*time.Millisecond)
	require.Equal(t, want, recv(t, ch, n))
}

// An open but silent producer must not block shutdown.
func TestConsumeStopsWithIdleConn(t *testing.T) {
	got := make(chan []byte, 1)
	path, stop := startListener(t, Config{Handler: Copy(func(b []byte) { got <- b })})

	c := dial(t, path)
	_, err := c.Write([]byte("{}\n"))
	require.NoError(t, err)
	recv(t, got, 1)
	stop()
}

// Producers may flush a line in several writes.
func TestConsumeJoinsFragments(t *testing.T) {
	line := `{"event_type":"alert","payload_printable":"GET / HTTP/1.1"}`
	got := make(chan []byte, 1)
	path, _ := startListener(t, Config{Handler: Copy(func(b []byte) { got <- b })})

	c := dial(t, path)
	for i := 0; i < len(line); i += 7 {
		_, err := c.Write([]byte(line[i:min(i+7, len(line))]))
		require.NoError(t, err)
		time.Sleep(time.Millisecond)
	}
	_, err := c.Write([]byte("\n"))
	require.NoError(t, err)
	require.Equal(t, []string{line}, recv(t, got, 1))
}

// Accept errors must not fall through to a nil conn.
func TestConsumeReturnsWhenListenerClosed(t *testing.T) {
	l, err := NewListener(Config{Path: sockPath(t), Handler: func([]byte) {}})
	require.NoError(t, err)

	done := make(chan error, 1)
	go func() { done <- l.Consume(context.Background()) }()
	require.NoError(t, l.UnixListener.Close())

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("Consume did not return after listener close")
	}
}

// Oversized lines are reported and do not stop the listener.
func TestConsumeReportsTooLong(t *testing.T) {
	got := make(chan []byte, 1)
	errs := make(chan error, 1)
	path, _ := startListener(t, Config{
		MaxLineSize: 1024,
		Handler:     Copy(func(b []byte) { got <- b }),
		OnError:     func(err error) { errs <- err },
	})

	c := dial(t, path)
	_, err := c.Write([]byte(strings.Repeat("x", 2048) + "\n"))
	require.NoError(t, err)
	select {
	case e := <-errs:
		require.ErrorIs(t, e, bufio.ErrTooLong)
	case <-time.After(3 * time.Second):
		t.Fatal("oversized line not reported")
	}

	c = dial(t, path)
	_, err = c.Write([]byte("{}\n"))
	require.NoError(t, err)
	require.Equal(t, []string{"{}"}, recv(t, got, 1))
}

// Force must only replace stale sockets, never other files.
func TestForceKeepsRegularFile(t *testing.T) {
	path := sockPath(t)
	require.NoError(t, os.WriteFile(path, []byte("keep"), 0o600))

	_, err := NewListener(Config{Path: path, Force: true, Handler: func([]byte) {}})
	require.Error(t, err)
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "keep", string(b))
}

// A socket left behind by a crashed process is replaced on restart.
func TestForceReplacesStaleSocket(t *testing.T) {
	path := sockPath(t)
	stale, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	require.NoError(t, err)
	stale.SetUnlinkOnClose(false)
	require.NoError(t, stale.Close())

	l, err := NewListener(Config{Path: path, Force: true, Handler: func([]byte) {}})
	require.NoError(t, err)
	require.NoError(t, l.UnixListener.Close())
}

// Persistent accept errors (EMFILE) must not spin, and serving resumes once they clear.
func TestConsumeBacksOffOnAcceptError(t *testing.T) {
	var errCount atomic.Int64
	got := make(chan []byte, 1)
	path, _ := startListener(t, Config{
		Handler: Copy(func(b []byte) { got <- b }),
		OnError: func(error) { errCount.Add(1) },
	})
	// client fd is created up front; connecting needs no new fd
	fd, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	require.NoError(t, err)
	t.Cleanup(func() { syscall.Close(fd) })

	var lim syscall.Rlimit
	require.NoError(t, syscall.Getrlimit(syscall.RLIMIT_NOFILE, &lim))
	orig := lim
	lim.Cur = 256
	require.NoError(t, syscall.Setrlimit(syscall.RLIMIT_NOFILE, &lim))
	var fill []*os.File
	for {
		f, ferr := os.Open(os.DevNull)
		if ferr != nil {
			break
		}
		fill = append(fill, f)
	}
	require.NoError(t, syscall.Connect(fd, &syscall.SockaddrUnix{Name: path}))
	time.Sleep(250 * time.Millisecond)
	for _, f := range fill {
		f.Close()
	}
	require.NoError(t, syscall.Setrlimit(syscall.RLIMIT_NOFILE, &orig))

	require.Positive(t, errCount.Load())
	require.Less(t, errCount.Load(), int64(20))
	_, err = syscall.Write(fd, []byte("{}\n"))
	require.NoError(t, err)
	require.Equal(t, []string{"{}"}, recv(t, got, 1))
}

// Copy must detach the kept line from the buffer the caller reuses.
func TestCopyDetachesFromBuffer(t *testing.T) {
	var kept []byte
	h := Copy(func(b []byte) { kept = b })
	buf := []byte(`{"event_type":"alert"}`)
	h(buf)
	copy(buf, `{"event_type":"flow"} `)
	require.Equal(t, `{"event_type":"alert"}`, string(kept))
}

// Without Copy, lines kept past the handler call are overwritten by later reads.
func TestRawHandlerLinesAreOverwritten(t *testing.T) {
	const n = 1000
	want := make([]string, n)
	for i := range want {
		want[i] = fmt.Sprintf(`{"seq":%d,"pad":%q}`, i, strings.Repeat("x", 64+i%97))
	}
	var kept [][]byte
	all := make(chan struct{})
	path, stop := startListener(t, Config{Handler: func(b []byte) {
		if kept = append(kept, b); len(kept) == n {
			close(all)
		}
	}})

	c := dial(t, path)
	_, err := c.Write([]byte(strings.Join(want, "\n") + "\n"))
	require.NoError(t, err)
	select {
	case <-all:
	case <-time.After(3 * time.Second):
		t.Fatal("lines not received")
	}
	// kept is only read once Consume has returned
	stop()
	got := make([]string, n)
	for i, b := range kept {
		got[i] = string(b)
	}
	require.NotEqual(t, want, got)
}

// Alerts with large printable payloads exceed bufio's 64 KiB default and must pass with default config.
func TestConsumeLargeAlert(t *testing.T) {
	body := strings.Repeat("HTTP/1.1 200 OK\r\nContent-Type: \"text/html\"\r\n\r\n<html>\x01\x7f</html>\n", 8192)
	alert, err := json.Marshal(map[string]any{
		"event_type":        "alert",
		"alert":             map[string]any{"signature_id": 2100498, "signature": "GPL ATTACK_RESPONSE id check returned root"},
		"payload_printable": body,
		"http":              map[string]any{"http_response_body_printable": body},
	})
	require.NoError(t, err)
	require.Greater(t, len(alert), 16*bufio.MaxScanTokenSize)

	lines := []string{`{"event_type":"flow"}`, string(alert), `{"event_type":"dns"}`}
	ch := make(chan []byte, len(lines))
	path, _ := startListener(t, Config{Handler: Copy(func(b []byte) { ch <- b })})

	c := dial(t, path)
	_, err = c.Write([]byte(strings.Join(lines, "\n") + "\n"))
	require.NoError(t, err)
	require.Equal(t, lines, recv(t, ch, len(lines)))
}

// MaxLineSize is the longest accepted line, excluding the newline.
func TestMaxLineSizeIsInclusive(t *testing.T) {
	const size = 1024
	got := make(chan []byte, 1)
	errs := make(chan error, 1)
	path, _ := startListener(t, Config{
		MaxLineSize: size,
		Handler:     Copy(func(b []byte) { got <- b }),
		OnError:     func(err error) { errs <- err },
	})

	c := dial(t, path)
	_, err := c.Write([]byte(strings.Repeat("x", size) + "\n"))
	require.NoError(t, err)
	select {
	case b := <-got:
		require.Len(t, b, size)
	case e := <-errs:
		t.Fatalf("line of MaxLineSize rejected: %v", e)
	case <-time.After(3 * time.Second):
		t.Fatal("line not received")
	}
}
