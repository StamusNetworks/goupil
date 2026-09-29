package socket

import "sync/atomic"

type Stats struct {
	Connections uint64
	Lines       uint64
	// Bytes counts line content, newlines excluded.
	Bytes        uint64
	AcceptErrors uint64
	ReadErrors   uint64
}

type counters struct {
	connections  atomic.Uint64
	lines        atomic.Uint64
	bytes        atomic.Uint64
	acceptErrors atomic.Uint64
	readErrors   atomic.Uint64
}

// Stats is safe to call while Consume runs.
func (l *Listener) Stats() Stats {
	return Stats{
		Connections:  l.stats.connections.Load(),
		Lines:        l.stats.lines.Load(),
		Bytes:        l.stats.bytes.Load(),
		AcceptErrors: l.stats.acceptErrors.Load(),
		ReadErrors:   l.stats.readErrors.Load(),
	}
}
